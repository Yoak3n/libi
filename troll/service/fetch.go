package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"troll/internal/util"
	"troll/model"

	"github.com/Yoak3n/libi/shared/domain/model/schema"
	"github.com/Yoak3n/libi/shared/domain/model/table"
)

const (
	QueryUrl      = "https://api.bilibili.com/x/v2/reply"
	SubReplyUrl   = "https://api.bilibili.com/x/v2/reply/reply"
	LazilyLoadUrl = "https://api.bilibili.com/x/v2/reply/wbi/main"
)

func LazilyGetAllComments(avid uint, total int, tracker *ProgressTracker, workerID int, title string) []model.CommentData {
	allComments := make([]model.CommentData, 0)
	offset := ""
	var counter atomic.Int64
	stopProgress := tracker.StartJob(workerID, &counter, total, title)
	for {
		params := map[string]string{
			"oid":          strconv.FormatUint(uint64(avid), 10),
			"type":         "1",
			"mode":         "3",
			"plat":         "1",
			"web_location": "1315875",
		}
		if offset != "" {
			params["pagination_str"] = url.QueryEscape(fmt.Sprintf(`{"offset":"%s"}`, offset))
		}
		uri := util.AppendParamsToUrl(LazilyLoadUrl, params)

		var resBuf []byte
		if err := dispatcher.Dispatch(func(cookie string) error {
			resBuf = util.RequestGetWithAll(uri, cookie)
			if resBuf == nil {
				return errors.New("nil response")
			}
			return nil
		}); err != nil {
			log.Printf("ERROR: LazilyGetAllComments: %v", err)
			continue
		}

		response := &model.LazyCommentResponse{}
		if err := json.Unmarshal(resBuf, response); err != nil || response.Code != 0 {
			log.Printf("ERROR: LazilyGetAllComments decode: %v code=%d", err, response.Code)
			continue
		}
		if response.Data.Cursor.IsEnd || len(response.Data.Replies) < 1 {
			break
		}
		currentComments := extractComments(response.Data.Replies, 0, &counter)
		for i, v := range currentComments {
			if v.NeedExpand {
				currentComments[i] = *getCommentSubTree(&v, &counter)
			}
		}
		allComments = append(allComments, currentComments...)
		offset = response.Data.Cursor.PaginationReply.NextOffset
	}
	stopProgress()
	return allComments
}

// ProgressTracker manages multiple worker progress bars on a shared terminal display.
// A single display goroutine renders all active workers atomically, avoiding interleaving
// with other terminal output (e.g. log.Printf).
type ProgressTracker struct {
	mu        sync.Mutex
	workers   map[int]*progressWorker
	lineWidth int // length of last rendered line, used to pad/overwrite
}

type progressWorker struct {
	counter  *atomic.Int64
	total    int
	title    string
	start    time.Time
	done     chan struct{}
	elapsed  time.Duration // set when worker finishes
	finished bool
}

func NewProgressTracker() *ProgressTracker {
	return &ProgressTracker{
		workers: make(map[int]*progressWorker),
	}
}

// RegisterWorker creates a persistent slot for a worker. Call once per worker lifetime.
func (pt *ProgressTracker) RegisterWorker(workerID int) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	if _, exists := pt.workers[workerID]; !exists {
		pt.workers[workerID] = &progressWorker{
			done: make(chan struct{}),
		}
	}
}

// StartJob begins a new job for the given worker. Returns a stop function.
func (pt *ProgressTracker) StartJob(workerID int, counter *atomic.Int64, total int, title string) func() {
	pt.mu.Lock()
	pw := pt.workers[workerID]
	pw.counter = counter
	pw.total = total
	pw.title = title
	pw.start = time.Now()
	pw.finished = false
	pw.elapsed = 0
	// Reset the done channel for this job
	pw.done = make(chan struct{})
	done := pw.done
	pt.mu.Unlock()

	go pt.displayLoop(done)

	return func() {
		close(done)
		pw.elapsed = time.Since(pw.start)
		pw.finished = true
		count := pw.counter.Load()
		if count > 0 {
			fmt.Printf("\r\033[2K  %s Collected %d comments. Elapsed: %v\n", pw.title, count, pw.elapsed.Round(time.Millisecond))
		}
		pt.lineWidth = 0
	}
}

func (pt *ProgressTracker) displayLoop(done chan struct{}) {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			pt.render()
		}
	}
}

func (pt *ProgressTracker) render() {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	if len(pt.workers) == 0 {
		return
	}
	// Find the max worker ID to know how many workers to render
	maxID := 0
	for id := range pt.workers {
		if id > maxID {
			maxID = id
		}
	}
	// Build all worker segments and print on a single line with \r
	var parts []string
	for i := 1; i <= maxID; i++ {
		pw, ok := pt.workers[i]
		if !ok {
			continue
		}
		count := pw.counter.Load()
		title := pw.title
		if len(title) > 20 {
			title = title[:17] + "..."
		}
		var seg string
		if pw.finished {
			seg = fmt.Sprintf("[%s] Done(%d, %v)", title, count, pw.elapsed.Round(time.Millisecond))
		} else if pw.total > 0 {
			pct := float64(count) / float64(pw.total) * 100
			barWidth := 15
			filled := int(float64(barWidth) * float64(count) / float64(pw.total))
			if filled > barWidth {
				filled = barWidth
			}
			bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
			elapsed := time.Since(pw.start).Seconds()
			speed := 0.0
			if elapsed > 0 {
				speed = float64(count) / elapsed
			}
			seg = fmt.Sprintf("[%s] %s %d/%d %.0f%% %.0f/s", bar, title, count, pw.total, pct, speed)
		} else {
			seg = fmt.Sprintf("%s %d", title, count)
		}
		parts = append(parts, seg)
	}
	line := "  " + strings.Join(parts, "  |  ")
	// Pad with spaces to clear any leftover characters from previous (longer) line
	if pt.lineWidth > len(line) {
		line += strings.Repeat(" ", pt.lineWidth-len(line))
	}
	pt.lineWidth = len(line)
	fmt.Printf("\r%s", line)
}

func extractComments(items []model.CommentItem, parent uint, counter *atomic.Int64) []model.CommentData {
	comments := make([]model.CommentData, 0, len(items))
	var commentRecords []*table.CommentTable
	var userRecords []*schema.User

	for _, v := range items {
		createdAt := time.Unix(v.Ctime, 0)
		author := model.UserData{
			Uid:      v.Mid,
			Name:     v.Member.Uname,
			Location: v.ReplyControl.Location,
		}
		userRecords = append(userRecords, &schema.User{
			UID:    v.Mid,
			Name:   v.Member.Uname,
			Avatar: v.Member.Avatar,
			Sex:    util.TransSex(v.Member.Sex),
		})
		comment := model.CommentData{
			Text:      v.Content.Message,
			Author:    author,
			Rpid:      v.Rpid,
			Oid:       v.Oid,
			Like:      v.Like,
			CreatedAt: createdAt,
		}
		commentRecord := &table.CommentTable{
			Text:        comment.Text,
			Owner:       comment.Author.Uid,
			VideoAvid:   comment.Oid,
			CommentId:   comment.Rpid,
			Like:        v.Like,
			CommentTime: createdAt,
		}
		if parent == 0 {
			comment.NeedExpand = v.ReplyControl.SubReplyEntryText != ""
			if !comment.NeedExpand {
				// Only persist inline sub-replies when there's no expansion needed.
				// When NeedExpand is true, getCommentSubTree will fetch the full tree
				// starting from page 1, which overlaps with v.Replies and causes
				// duplicate primary key failures that lose the entire batch.
				comment.Children = extractComments(v.Replies, v.Rpid, counter)
			}
		} else {
			commentRecord.ParentComment = parent
		}
		commentRecords = append(commentRecords, commentRecord)
		comments = append(comments, comment)
		counter.Add(1)
	}

	go func() {
		if len(userRecords) > 0 {
			UserRepo.CreateOrUpdateUserBatch(userRecords)
		}
	}()
	go func() {
		if len(commentRecords) > 0 {
			CommentRepo.CreateCommentBatch(commentRecords)
		}
	}()
	return comments
}

func getCommentSubTree(comment *model.CommentData, counter *atomic.Int64) *model.CommentData {
	page := 1
	subComments := make([]model.CommentData, 0)
	times := 0
	for {
		if times >= 5 {
			break
		}
		params := map[string]string{
			"type": "1",
			"oid":  strconv.FormatUint(uint64(comment.Oid), 10),
			"root": strconv.FormatUint(uint64(comment.Rpid), 10),
			"ps":   "20",
			"pn":   strconv.Itoa(page),
		}
		uri := util.AppendParamsToUrl(SubReplyUrl, params)

		var resBuf []byte
		if err := dispatcher.Dispatch(func(cookie string) error {
			resBuf = util.RequestGetWithAll(uri, cookie)
			if resBuf == nil {
				return errors.New("nil response")
			}
			return nil
		}); err != nil {
			times++
			continue
		}
		response := &model.SubCommentResponse{}
		if err := json.Unmarshal(resBuf, response); err != nil || response.Code != 0 {
			times++
			continue
		}
		if len(response.Data.Replies) < 1 {
			break
		}
		replies := extractComments(response.Data.Replies, comment.Rpid, counter)
		subComments = append(subComments, replies...)
		page++
	}
	comment.Children = subComments
	return comment
}

func FetchVideoComments(avid uint, offset string) ([]model.CommentData, int, string) {
	params := map[string]string{
		"oid":          strconv.FormatUint(uint64(avid), 10),
		"type":         "1",
		"mode":         "3",
		"plat":         "1",
		"web_location": "1315875",
	}
	if offset != "" {
		params["pagination_str"] = url.QueryEscape(fmt.Sprintf(`{"offset":"%s"}`, offset))
	}
	uri := util.AppendParamsToUrl(LazilyLoadUrl, params)

	var resBuf []byte
	if err := dispatcher.Dispatch(func(cookie string) error {
		resBuf = util.RequestGetWithAll(uri, cookie)
		if resBuf == nil {
			return errors.New("nil response")
		}
		return nil
	}); err != nil {
		return nil, 0, ""
	}

	response := &model.LazyCommentResponse{}
	if err := json.Unmarshal(resBuf, response); err != nil || response.Code != 0 {
		return nil, 0, ""
	}
	if response.Data.Cursor.IsEnd || len(response.Data.Replies) < 1 {
		return nil, 0, response.Data.Cursor.PaginationReply.NextOffset
	}

	var counter atomic.Int64
	comments := extractComments(response.Data.Replies, 0, &counter)
	for i, v := range comments {
		if v.NeedExpand {
			comments[i] = *getCommentSubTree(&v, &counter)
		}
	}
	return comments, int(counter.Load()), response.Data.Cursor.PaginationReply.NextOffset
}

func SearchVideoOfTopic(keyword string, page int) []model.VideoData {
	const SearchUrl = "https://api.bilibili.com/x/web-interface/wbi/search/type"
	params := map[string]string{
		"search_type": "video",
		"keyword":     keyword,
		"page":        strconv.Itoa(page),
		"order":       "totalrank",
	}
	addr := util.AppendParamsToUrl(SearchUrl, params)

	var resBuf []byte
	if err := dispatcher.Dispatch(func(cookie string) error {
		resBuf = util.RequestGetWithAll(addr, cookie)
		if resBuf == nil {
			return errors.New("nil response")
		}
		return nil
	}); err != nil {
		return nil
	}

	response := model.SearchResponse{}
	if err := json.Unmarshal(resBuf, &response); err != nil || response.Code != 0 {
		return nil
	}
	videos := make([]model.VideoData, 0, len(response.Data.Result))
	for _, v := range response.Data.Result {
		videos = append(videos, model.VideoData{
			Avid:        v.Aid,
			Bvid:        v.Bvid,
			Title:       util.ExtractContentWithinTag(v.Title),
			Cover:       v.Pic,
			Description: v.Description,
			Owner:       model.UserData{Uid: v.Mid, Name: v.Author},
			Review:      v.Review,
		})
	}
	return videos
}

func FetchVideoInfo(bvid string, topic string) *model.VideoData {
	const VideoDetailUrl = "https://api.bilibili.com/x/web-interface/wbi/view/detail"
	params := map[string]string{"bvid": bvid}
	addr := util.AppendParamsToUrl(VideoDetailUrl, params)

	var resBuf []byte
	if err := dispatcher.Dispatch(func(cookie string) error {
		resBuf = util.RequestGetWithAll(addr, cookie)
		if resBuf == nil {
			return errors.New("nil response")
		}
		return nil
	}); err != nil {
		return nil
	}

	resp := model.VideoDetailResponse{}
	if err := json.Unmarshal(resBuf, &resp); err != nil || resp.Code != 0 {
		return nil
	}

	v := resp.Data.View
	tags := ""
	if len(resp.Data.Tags) > 0 {
		names := make([]string, 0, len(resp.Data.Tags))
		for _, t := range resp.Data.Tags {
			if t.TagName != "" {
				names = append(names, t.TagName)
			}
		}
		tags = strings.Join(names, ", ")
	}

	ret := &model.VideoData{
		Avid:        v.Aid,
		Bvid:        v.Bvid,
		Title:       v.Title,
		Cover:       v.Pic,
		Description: v.Description,
		Owner:       model.UserData{Uid: v.Owner.Mid, Name: v.Owner.Name},
		Review:      int(v.Stat.Reply),
		Tags:        tags,
	}
	videoRecord := &table.VideoTable{
		Avid: ret.Avid, Title: ret.Title, Bvid: ret.Bvid,
		Description: ret.Description, Owner: ret.Owner.Uid, Topic: topic,
		Tags: tags,
	}
	VideoRepo.CreateVideo(videoRecord)
	AddUserByUid(ret.Owner.Uid)
	return ret
}

func AddUserByUid(uid uint) {
	const UserApi = "https://api.bilibili.com/x/space/wbi/acc/info"
	addr := util.AppendParamsToUrl(UserApi, map[string]string{
		"mid": fmt.Sprintf("%d", uid),
	})

	var resBuf []byte
	if err := dispatcher.Dispatch(func(cookie string) error {
		resBuf = util.RequestGetWithAll(addr, cookie)
		if resBuf == nil {
			return errors.New("nil response")
		}
		return nil
	}); err != nil {
		return
	}

	var resp model.UserResponse
	if err := json.Unmarshal(resBuf, &resp); err != nil || resp.Code != 0 {
		return
	}
	UserRepo.CreateOrUpdateUserBatch([]*schema.User{
		{UID: resp.Data.Mid, Name: resp.Data.Name, Avatar: resp.Data.Face, Sex: util.TransSex(resp.Data.Sex)},
	})
}

func CountComments(video *model.VideoData) uint {
	var count uint
	for _, c := range video.Comments {
		count++
		count += uint(len(c.Children))
	}
	return count
}

// GetCommentsByVideo wraps troll repo for CLI query
func GetCommentsByVideo(avid uint) schema.CommentGroupByVideo {
	return TrollRepo.GetCommentsByVideo(avid)
}

// QueryTopNUser wraps troll repo
func QueryTopNUser(topic string, n int) ([]schema.UserQuery, error) {
	return TrollRepo.QueryTopNUserInTopic(topic, n)
}

// QueryTopNUserMultiTopic wraps troll repo for multi-topic query
func QueryTopNUserMultiTopic(topics []string, n int) ([]schema.UserQuery, error) {
	return TrollRepo.QueryTopNUserInTopics(topics, n)
}

// QuerySimilarComments wraps troll repo
func QuerySimilarComments(topic string, n int) ([]schema.SimilarCommentGroup, error) {
	return TrollRepo.QuerySimilarComments(topic, n)
}

// GetVideosByTopic wraps troll repo
func GetVideosByTopic(topic string) []schema.VideoWithCommentCount {
	return TrollRepo.GetVideosByTopic(topic)
}

// GetAllTopicsList wraps troll repo
func GetAllTopicsList() []schema.TopicsData {
	return TrollRepo.GetAllTopicsList()
}

// SearchCommentWithKeyword wraps troll repo
func SearchCommentWithKeyword(keyword string) []schema.CommentData {
	return TrollRepo.SearchCommentWithKeyword(keyword)
}

// GetDashboardStats wraps troll repo
func GetDashboardStats() schema.DashboardStats {
	return TrollRepo.GetDashboardStats()
}

// GetUserCommentsInTopic wraps troll repo
func GetUserCommentsInTopic(uid uint, topic string) []schema.CommentGroupByVideo {
	return TrollRepo.GetCommentsWithVideoFromUserInTopic(uid, topic)
}

// FetchVideoInfoForCLI is a wrapper that handles video info fetch with topic
func FetchVideoInfoForCLI(bvid string, avid int64, topic string) *model.VideoData {
	if avid != -1 {
		bvid = util.Avid2Bvid(avid)
	}
	return FetchVideoInfo(bvid, topic)
}

var ErrNoCookie = errors.New("no valid cookie found")
