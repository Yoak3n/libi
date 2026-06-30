package pipeline

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"gorm.io/gorm"
	agent "mirror/internal/agent"
	mirrorconfig "mirror/internal/config"
	"mirror/internal/store"
)

// RunAnalysis 完整流水线：
// 1. 从 Bilibili API 搜索视频 → 写入 shared DB
// 2. 拉取评论 → 写入 shared DB
// 3. 从 shared DB 读取 → Agent 分析 → JSON 文件
func RunAnalysis(ctx context.Context, taskName string, communities []string) error {
	log.Printf("[pipeline] RunAnalysis: %s communities=%v", taskName, communities)

	db := store.DB()
	if db == nil {
		return fmt.Errorf("database not available")
	}

	// 0. 准备输出目录
	workDir := "data"
	if mirrorconfig.Conf != nil && mirrorconfig.Conf.WorkDir != "" {
		workDir = mirrorconfig.Conf.WorkDir
	}
	outputDir, err := store.PrepareOutputDir(workDir, taskName)
	if err != nil {
		return fmt.Errorf("prepare output dir: %w", err)
	}
	log.Printf("[pipeline] output → %s", outputDir)

	// 1. 搜索 & 抓取
	videos, err := crawlAndSave(db, communities)
	if err != nil {
		return fmt.Errorf("crawl: %w", err)
	}
	if len(videos) == 0 {
		log.Printf("[pipeline] no videos found for %v", communities)
		return nil
	}

	// 2. 读取并分析
	videoTasks, err := LoadVideoTasks(communities, 50)
	if err != nil {
		return fmt.Errorf("load tasks: %w", err)
	}
	if len(videoTasks) == 0 {
		return nil
	}

	o := agent.NewOrchestrator(taskName, communities)
	o.SetOutputDir(outputDir)
	for _, v := range videoTasks {
		o.AddVideo(v)
	}
	if err := o.Run(ctx); err != nil {
		return err
	}

	log.Printf("[pipeline] complete → %s (%d videos)", outputDir, len(videos))
	return nil
}

// RunAnalysisWithVideoList 从指定视频列表文件抓取并分析。
// 文件格式：每行一个 BVID 或 BVID=社区名
func RunAnalysisWithVideoList(ctx context.Context, taskName string, videoListPath string) error {
	db := store.DB()
	if db == nil {
		return fmt.Errorf("database not available")
	}

	// 读取视频列表
	data, err := os.ReadFile(videoListPath)
	if err != nil {
		return fmt.Errorf("read video list: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 0 {
		return fmt.Errorf("empty video list")
	}

	workDir := "data"
	if mirrorconfig.Conf != nil && mirrorconfig.Conf.WorkDir != "" {
		workDir = mirrorconfig.Conf.WorkDir
	}
	outputDir, err := store.PrepareOutputDir(workDir, taskName)
	if err != nil {
		return fmt.Errorf("prepare output dir: %w", err)
	}

	// 逐个抓取
	var videoTasks []agent.VideoTask
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 支持 BVID=社区名 格式
		bvid := line
		topic := ""
		if idx := strings.Index(line, "="); idx > 0 {
			bvid = strings.TrimSpace(line[:idx])
			topic = strings.TrimSpace(line[idx+1:])
		}

		log.Printf("[pipeline] fetching video %s (topic=%q)", bvid, topic)

		detail, err := FetchVideoDetail(bvid)
		if err != nil {
			log.Printf("[pipeline] skip %s: %v", bvid, err)
			continue
		}
		if topic == "" {
			topic = detail.Title
		}

		SaveVideo(db, detail, topic)
		SaveUser(db, detail.Owner.Mid, detail.Owner.Name, "")

		comments, err := FetchAllComments(detail.Aid, 5)
		if err != nil {
			log.Printf("[pipeline] comments %s failed: %v", bvid, err)
			continue
		}
		for _, c := range comments {
			SaveComment(db, c)
			SaveUser(db, c.Mid, c.Uname, "")
			for _, sub := range c.Replies {
				SaveComment(db, sub)
				SaveUser(db, sub.Mid, sub.Uname, "")
			}
		}

		// 组装 CommentInput
		var inputs []agent.CommentInput
		for _, c := range comments {
			inputs = append(inputs, agent.CommentInput{
				Rpid:                c.Rpid,
				CommentText:         c.Content,
				CommentAuthor:       c.Mid,
				AuthorName:          c.Uname,
				VideoTitle:          detail.Title,
				VideoAvid:           detail.Aid,
				VideoTags:           detail.Tags,
				AffiliatedCommunity: topic,
			})
		}
		videoTasks = append(videoTasks, agent.VideoTask{
			Avid:     detail.Aid,
			Bvid:     detail.Bvid,
			Title:    detail.Title,
			Comments: inputs,
		})
	}

	if len(videoTasks) == 0 {
		return fmt.Errorf("no videos could be fetched from list")
	}

	o := agent.NewOrchestrator(taskName, nil)
	o.SetOutputDir(outputDir)
	for _, v := range videoTasks {
		o.AddVideo(v)
	}
	return o.Run(ctx)
}

// crawlAndSave 搜索视频→拉评论→存 DB，返回视频列表
func crawlAndSave(db *gorm.DB, communities []string) ([]SearchResult, error) {
	// 收集搜索关键词
	var keywords []string
	for _, name := range communities {
		keywords = append(keywords, name)
		if def := mirrorconfig.FindCommunity(name); def != nil {
			keywords = append(keywords, def.Aliases...)
		}
	}

	var allVideos []SearchResult
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 3) // 并发 3

	for _, kw := range keywords {
		wg.Add(1)
		sem <- struct{}{}
		go func(keyword string) {
			defer wg.Done()
			defer func() { <-sem }()

			results, err := SearchVideos(keyword, 1)
			if err != nil {
				log.Printf("[crawl] search %q failed: %v", keyword, err)
				return
			}
			log.Printf("[crawl] search %q → %d results", keyword, len(results))

			mu.Lock()
			allVideos = append(allVideos, results...)
			mu.Unlock()

			// 逐个视频拉详情 + 评论
			for _, v := range results {
				wg.Add(1)
				sem <- struct{}{}
				go func(sr SearchResult) {
					defer wg.Done()
					defer func() { <-sem }()

					// 视频详情
					detail, err := FetchVideoDetail(sr.Bvid)
					if err != nil {
						log.Printf("[crawl] detail %s failed: %v", sr.Bvid, err)
						return
					}
					// 确定归属社区
					topic := resolveAffiliatedCommunity(sr.Bvid, communities)
					if topic == "" {
						topic = communities[0]
					}
					SaveVideo(db, detail, topic)
					SaveUser(db, detail.Owner.Mid, detail.Owner.Name, "")

					// 评论
					comments, err := FetchAllComments(detail.Aid, 5)
					if err != nil {
						log.Printf("[crawl] comments %s failed: %v", sr.Bvid, err)
						return
					}
					for _, c := range comments {
						SaveComment(db, c)
						SaveUser(db, c.Mid, c.Uname, "")
						for _, sub := range c.Replies {
							SaveComment(db, sub)
							SaveUser(db, sub.Mid, sub.Uname, "")
						}
					}
					log.Printf("[crawl] %s → %d comments saved", sr.Bvid, len(comments))
				}(v)
			}
		}(kw)
	}
	wg.Wait()

	log.Printf("[crawl] total %d videos fetched", len(allVideos))
	return allVideos, nil
}
