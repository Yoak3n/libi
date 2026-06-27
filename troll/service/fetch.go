package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
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

func LazilyGetAllComments(avid uint, total int) []model.CommentData {
	allComments := make([]model.CommentData, 0)
	offset := ""
	var counter atomic.Int64
	stopProgress := startProgressBar(&counter, total)
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

func startProgressBar(counter *atomic.Int64, total int) func() {
	start := time.Now()
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				count := counter.Load()
				if total > 0 {
					pct := float64(count) / float64(total) * 100
					barWidth := 30
					filled := int(float64(barWidth) * float64(count) / float64(total))
					if filled > barWidth {
						filled = barWidth
					}
					bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
					fmt.Printf("  [%s] %d/%d %.1f%%\r", bar, count, total, pct)
				} else {
					fmt.Printf("  Collecting comments: %d\r", count)
				}
			}
		}
	}()
	return func() {
		close(done)
		elapsed := time.Since(start)
		count := counter.Load()
		if count > 0 {
			fmt.Printf("  Collected %d comments. Elapsed: %v          \n", count, elapsed.Round(time.Millisecond))
		}
	}
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
	const VideoInfoUrl = "https://api.bilibili.com/x/web-interface/wbi/view"
	params := map[string]string{"bvid": bvid}
	addr := util.AppendParamsToUrl(VideoInfoUrl, params)

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

	response := model.VideoInfoResponse{}
	if err := json.Unmarshal(resBuf, &response); err != nil || response.Code != 0 {
		return nil
	}
	v := response.Data
	ret := &model.VideoData{
		Avid:        v.Aid,
		Bvid:        v.Bvid,
		Title:       v.Title,
		Cover:       v.Pic,
		Description: v.Description,
		Owner:       model.UserData{Uid: v.Owner.Mid, Name: v.Owner.Name},
		Review:      int(v.Stat.Reply),
	}
	videoRecord := &table.VideoTable{
		Avid: ret.Avid, Title: ret.Title, Bvid: ret.Bvid,
		Description: ret.Description, Owner: ret.Owner.Uid, Topic: topic,
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

// QuerySimilarComments wraps troll repo
func QuerySimilarComments(topic string, n int) ([]schema.SimilarCommentResult, error) {
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
