package schema

import "time"

type SimilarCommentResult struct {
	Text       string
	Count      int
	CommentIds string
	Example1   string
	Example2   string
}

type UserQuery struct {
	UID      uint   `json:"uid"`
	Username string `json:"username"`
	Avatar   string `json:"avatar"`
	Location string
	Count    int
}

type CommentData struct {
	Id        uint          `json:"id"`
	Content   string        `json:"content"`
	Like      uint64        `json:"like"`
	Owner     Author        `json:"owner"`
	CreatedAt time.Time     `json:"created_at"`
	Children  []CommentData `json:"children,omitempty"`
}

type Author struct {
	Uid      uint   `json:"uid"`
	Name     string `json:"name"`
	Avatar   string `json:"avatar"`
	Location string `json:"location"`
}

type VideoData struct {
	Avid        uint   `json:"avid"`
	Bvid        string `json:"bvid"`
	Title       string `json:"title"`
	Topic       string `json:"topic"`
	Description string `json:"description"`
	Cover       string `json:"cover"`
	Author      Author `json:"author"`
}

type VideoWithCommentCount struct {
	Count    int    `json:"count"`
	UpdateAt string `json:"update_at"`
	VideoData
}

type CommentGroupByVideo struct {
	VideoData
	Comments []CommentData `json:"comments"`
}

type TopicsData struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

type DashboardStats struct {
	Topics   int64 `json:"topics"`
	Videos   int64 `json:"videos"`
	Users    int64 `json:"users"`
	Comments int64 `json:"comments"`
}
