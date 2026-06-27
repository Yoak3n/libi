package model

import "time"

type VideoDataOutput struct {
	VideoID string     `json:"video_id"`
	Count   uint       `json:"count"`
	Data    *VideoData `json:"data,omitempty"`
}

type VideoData struct {
	Avid        uint          `json:"avid"`
	Bvid        string        `json:"bvid"`
	Title       string        `json:"title"`
	Cover       string        `json:"cover"`
	Description string        `json:"description"`
	Owner       UserData      `json:"owner"`
	Comments    []CommentData `json:"comments"`
	Review      int           `json:"review"`
}

type CommentData struct {
	Text       string        `json:"text"`
	Author     UserData      `json:"author"`
	Children   []CommentData `json:"children"`
	Rpid       uint          `json:"rpid"`
	Oid        uint          `json:"oid"`
	Like       uint64        `json:"like"`
	NeedExpand bool
	CreatedAt  time.Time     `json:"created_at"`
}

type UserData struct {
	Uid      uint   `json:"uid"`
	Name     string `json:"name"`
	Location string `json:"location"`
	Avatar   string `json:"avatar"`
}
