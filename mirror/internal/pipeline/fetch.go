package pipeline

import (
	"fmt"
	"log"

	"github.com/Yoak3n/libi/shared/domain/model/table"
	"gorm.io/gorm"

	agent "mirror/internal/agent"
	mirrorconfig "mirror/internal/config"
	"mirror/internal/store"
)

// LoadVideoTasks 从 troll 的数据库读取视频及其评论。
//
// 社区归属逻辑：troll 在 fetch 时会将搜索关键词存入 video_tables.topic 字段。
// 例如 `troll fetch -t "原神"` 会生成 topic="原神" 的视频记录。
// 我们直接按 mirror.yaml 中定义的社区名/别名搜索 topic 字段。
func LoadVideoTasks(communityNames []string, maxVideos int) ([]agent.VideoTask, error) {
	db := store.DB()
	if db == nil {
		return nil, fmt.Errorf("database not available (need troll's config.yaml)")
	}

	// 1. 收集所有搜索关键词（社区名 + 别名）
	var keywords []string
	for _, name := range communityNames {
		keywords = append(keywords, name)
		if def := mirrorconfig.FindCommunity(name); def != nil {
			keywords = append(keywords, def.Aliases...)
		}
	}

	// 2. 查 video_tables：topic 匹配任一关键词
	var videoRecords []table.VideoTable
	query := db.Where("topic IN ?", keywords).
		Limit(maxVideos).
		Order("created_at DESC").
		Find(&videoRecords)
	if query.Error != nil {
		return nil, fmt.Errorf("query videos: %w", query.Error)
	}
	if len(videoRecords) == 0 {
		log.Printf("[pipeline] no videos found for %v (topics: %v)", communityNames, keywords)
		log.Printf("[pipeline] make sure troll has fetched data: troll fetch -t \"原神\"")
		return nil, nil
	}

	// 3. 逐视频加载评论
	var tasks []agent.VideoTask
	for _, vr := range videoRecords {
		comments := loadCommentsForVideo(db, vr.Avid)
		if len(comments) == 0 {
			continue
		}

		// 确定视频归属社区（优先用 alias 映射到标准名）
		affiliated := resolveAffiliatedCommunity(vr.Topic, communityNames)

		inputs := make([]agent.CommentInput, 0, len(comments))
		for _, c := range comments {
			inputs = append(inputs, agent.CommentInput{
				Rpid:                c.CommentId,
				CommentText:         c.Text,
				CommentAuthor:       c.Owner,
				AuthorName:          resolveUserName(db, c.Owner),
				VideoTitle:          vr.Title,
				VideoAvid:           vr.Avid,
				VideoTags:           vr.Tags,
				AffiliatedCommunity: affiliated,
			})
		}

		tasks = append(tasks, agent.VideoTask{
			Avid:     vr.Avid,
			Bvid:     vr.Bvid,
			Title:    vr.Title,
			Comments: inputs,
		})

		log.Printf("[pipeline] video %s topic=%q → community=%q (%d comments)",
			vr.Bvid, vr.Topic, affiliated, len(inputs))
	}

	log.Printf("[pipeline] loaded %d videos for %v", len(tasks), communityNames)
	return tasks, nil
}

// resolveAffiliatedCommunity 将 video_tables 中的 topic 映射到标准社区名
func resolveAffiliatedCommunity(topic string, targets []string) string {
	if topic == "" {
		return ""
	}
	// 直接匹配
	for _, t := range targets {
		if topic == t {
			return t
		}
	}
	// 通过 alias 匹配
	for _, t := range targets {
		if def := mirrorconfig.FindCommunity(t); def != nil {
			for _, alias := range def.Aliases {
				if topic == alias {
					return t
				}
			}
		}
	}
	return topic // 未知 topic 原样返回
}

func loadCommentsForVideo(db *gorm.DB, avid uint) []table.CommentTable {
	var comments []table.CommentTable
	db.Where("video_avid = ? AND parent_comment = 0", avid).
		Order("comment_time ASC").
		Find(&comments)
	return comments
}

func resolveUserName(db *gorm.DB, uid uint) string {
	var user table.UserTable
	err := db.Where("uid = ?", uid).First(&user).Error
	if err != nil {
		return ""
	}
	return user.Name
}
