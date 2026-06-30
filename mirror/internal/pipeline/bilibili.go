// Package pipeline 提供 Bilibili API 调用，镜像 troll 的核心数据采集能力。
// 复用 shared/package/request 的 HTTP + WBI 签名，自实现请求调度。
package pipeline

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/Yoak3n/libi/shared/domain/model/table"
	"github.com/Yoak3n/libi/shared/package/request"
	"gorm.io/gorm"
)

// ──────────────────────────────────────────────
// 请求限流（简单版：固定间隔）
// ──────────────────────────────────────────────

var lastRequest time.Time

func rateLimit() {
	elapsed := time.Since(lastRequest)
	if elapsed < 2*time.Second {
		time.Sleep(2*time.Second - elapsed)
	}
	lastRequest = time.Now()
}

// ──────────────────────────────────────────────
// 搜索视频
// ──────────────────────────────────────────────

// SearchResult Bilibili 搜索返回的视频条目
type SearchResult struct {
	Aid         uint   `json:"aid"`
	Bvid        string `json:"bvid"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Pic         string `json:"pic"`
	Mid         uint   `json:"mid"`
	Author      string `json:"author"`
	Review      int    `json:"review"`
	Pubdate     int64  `json:"pubdate"`
	Tag         string `json:"tag"`
}

// SearchAnchor 搜索返回的分页锚点
type SearchAnchor struct {
	Score  string `json:"score"`
	PgInfo struct {
		Page int `json:"page"`
	} `json:"pg_info"`
}

// searchResponse Bilibili 搜索 API 响应
type searchResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Result []struct {
			Aid         uint   `json:"aid"`
			Bvid        string `json:"bvid"`
			Title       string `json:"title"`
			Description string `json:"description"`
			Pic         string `json:"pic"`
			Mid         uint   `json:"mid"`
			Author      string `json:"author"`
			Review      int    `json:"review"`
			Pubdate     int64  `json:"pubdate"`
			Tag         string `json:"tag"`
		} `json:"result"`
	} `json:"data"`
}

// SearchVideos 按关键词搜索视频，返回搜索结果
func SearchVideos(keyword string, page int) ([]SearchResult, error) {
	rateLimit()
	urlStr := fmt.Sprintf("https://api.bilibili.com/x/web-interface/wbi/search/type?search_type=video&keyword=%s&page=%d&order=totalrank", keyword, page)

	resp, err := request.GetWithWbi(urlStr)
	if err != nil {
		return nil, fmt.Errorf("search request: %w", err)
	}

	var res searchResponse
	if err := decodeJSON(resp, &res); err != nil {
		return nil, err
	}
	if res.Code != 0 {
		return nil, fmt.Errorf("search API error: code=%d msg=%s", res.Code, res.Message)
	}

	var results []SearchResult
	for _, r := range res.Data.Result {
		results = append(results, SearchResult{
			Aid:         r.Aid,
			Bvid:        r.Bvid,
			Title:       r.Title,
			Description: r.Description,
			Pic:         r.Pic,
			Mid:         r.Mid,
			Author:      r.Author,
			Review:      r.Review,
			Pubdate:     r.Pubdate,
		})
	}
	return results, nil
}

// ──────────────────────────────────────────────
// 视频详情
// ──────────────────────────────────────────────

// VideoDetail Bilibili 视频详情
type VideoDetail struct {
	Aid   uint   `json:"aid"`
	Bvid  string `json:"bvid"`
	Title string `json:"title"`
	Desc  string `json:"desc"`
	Pic   string `json:"pic"`
	Owner struct {
		Mid  uint   `json:"mid"`
		Name string `json:"name"`
	} `json:"owner"`
	Stat struct {
		Reply int `json:"reply"`
	} `json:"stat"`
	Pubdate int64  `json:"pubdate"`
	Tags    string `json:"tags"`
}

type videoDetailResponse struct {
	Code int          `json:"code"`
	Data *VideoDetail `json:"data"`
}

// FetchVideoDetail 获取视频详情
func FetchVideoDetail(bvid string) (*VideoDetail, error) {
	rateLimit()
	urlStr := fmt.Sprintf("https://api.bilibili.com/x/web-interface/wbi/view/detail?bvid=%s", bvid)

	resp, err := request.GetWithWbi(urlStr)
	if err != nil {
		return nil, fmt.Errorf("detail request: %w", err)
	}

	var res videoDetailResponse
	if err := decodeJSON(resp, &res); err != nil {
		return nil, err
	}
	if res.Code != 0 || res.Data == nil {
		return nil, fmt.Errorf("detail API error: code=%d", res.Code)
	}
	return res.Data, nil
}

// ──────────────────────────────────────────────
// 评论拉取
// ──────────────────────────────────────────────

// BiliComment Bilibili 评论条目
type BiliComment struct {
	Rpid    uint          `json:"rpid"`
	Oid     uint          `json:"oid"`
	Mid     uint          `json:"mid"`
	Uname   string        `json:"uname"`
	Content string        `json:"content"`
	Like    uint64        `json:"like"`
	Ctime   int64         `json:"ctime"`
	Replies []BiliComment `json:"replies,omitempty"`
}

type commentResponse struct {
	Code int `json:"code"`
	Data *struct {
		Replies []struct {
			Rpid   uint `json:"rpid"`
			Oid    uint `json:"oid"`
			Mid    uint `json:"mid"`
			Member struct {
				Uname  string `json:"uname"`
				Avatar string `json:"avatar"`
			} `json:"member"`
			Content struct {
				Message string `json:"message"`
			} `json:"content"`
			Like    uint64 `json:"like"`
			Ctime   int64  `json:"ctime"`
			Replies []struct {
				Rpid   uint `json:"rpid"`
				Mid    uint `json:"mid"`
				Member struct {
					Uname  string `json:"uname"`
					Avatar string `json:"avatar"`
				} `json:"member"`
				Content struct {
					Message string `json:"message"`
				} `json:"content"`
				Like  uint64 `json:"like"`
				Ctime int64  `json:"ctime"`
			} `json:"replies,omitempty"`
		} `json:"replies"`
		Cursor *struct {
			IsEnd           bool `json:"is_end"`
			PaginationReply struct {
				NextOffset string `json:"next_offset"`
			} `json:"pagination_reply"`
		} `json:"cursor"`
	} `json:"data"`
}

// FetchAllComments 拉取一个视频的所有评论（含子评论）
func FetchAllComments(avid uint, maxPages int) ([]BiliComment, error) {
	var all []BiliComment
	offset := ""

	for page := 0; page < maxPages; page++ {
		rateLimit()
		urlStr := fmt.Sprintf("https://api.bilibili.com/x/v2/reply/wbi/main?oid=%d&type=1&mode=3&plat=1&web_location=1315875", avid)
		if offset != "" {
			urlStr += "&pagination_str=" + urlQueryEscape(`{"offset":"`+offset+`"}`)
		}

		resp, err := request.GetWithWbi(urlStr)
		if err != nil {
			return nil, fmt.Errorf("comments request: %w", err)
		}

		var res commentResponse
		if err := decodeJSON(resp, &res); err != nil {
			return nil, err
		}
		if res.Code != 0 || res.Data == nil {
			break
		}
		if len(res.Data.Replies) == 0 || res.Data.Cursor.IsEnd {
			break
		}

		for _, r := range res.Data.Replies {
			comment := BiliComment{
				Rpid:    r.Rpid,
				Oid:     avid,
				Mid:     r.Mid,
				Uname:   r.Member.Uname,
				Content: r.Content.Message,
				Like:    r.Like,
				Ctime:   r.Ctime,
			}
			for _, sub := range r.Replies {
				comment.Replies = append(comment.Replies, BiliComment{
					Rpid:    sub.Rpid,
					Oid:     avid,
					Mid:     sub.Mid,
					Uname:   sub.Member.Uname,
					Content: sub.Content.Message,
					Like:    sub.Like,
					Ctime:   sub.Ctime,
				})
			}
			all = append(all, comment)
		}
		// offset 由下方的 Cursor 更新
		if res.Data.Cursor != nil {
			offset = res.Data.Cursor.PaginationReply.NextOffset
		} else {
			break
		}
	}

	return all, nil
}

// ──────────────────────────────────────────────
// 数据持久化（写入 shared DB 表）
// ──────────────────────────────────────────────

// SaveVideo 保存视频到 video_tables
func SaveVideo(db *gorm.DB, detail *VideoDetail, topic string) {
	record := &table.VideoTable{
		Avid:        detail.Aid,
		Bvid:        detail.Bvid,
		Title:       detail.Title,
		Cover:       detail.Pic,
		Topic:       topic,
		Description: detail.Desc,
		Tags:        detail.Tags,
		Owner:       detail.Owner.Mid,
	}
	db.Where("avid = ?", detail.Aid).Assign(record).FirstOrCreate(&table.VideoTable{Avid: detail.Aid})
}

// SaveComment 保存评论到 comment_tables
func SaveComment(db *gorm.DB, c BiliComment) {
	record := &table.CommentTable{
		CommentId:   c.Rpid,
		Text:        c.Content,
		Owner:       c.Mid,
		VideoAvid:   c.Oid,
		Like:        c.Like,
		CommentTime: time.Unix(c.Ctime, 0),
	}
	db.Where("comment_id = ?", c.Rpid).Assign(record).FirstOrCreate(&table.CommentTable{CommentId: c.Rpid})
}

// SaveUser 保存用户到 user_tables
func SaveUser(db *gorm.DB, uid uint, name, avatar string) {
	record := &table.UserTable{
		UID:    uid,
		Name:   name,
		Avatar: avatar,
	}
	db.Where("uid = ?", uid).Assign(record).FirstOrCreate(&table.UserTable{UID: uid})
}

// ──────────────────────────────────────────────
// 工具函数
// ──────────────────────────────────────────────

func decodeJSON(data []byte, v interface{}) error {
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("json decode: %w", err)
	}
	return nil
}

func urlQueryEscape(s string) string {
	return url.PathEscape(s)
}
