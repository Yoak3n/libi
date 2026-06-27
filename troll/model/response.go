package model

type ResponseCommon struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Comment API responses

type CommentResponse struct {
	ResponseCommon `json:"-"`
	Data           CommentResponseData `json:"data"`
}

type CommentResponseData struct {
	Replies []CommentItem `json:"replies"`
}

type LazyCommentResponse struct {
	ResponseCommon `json:"-"`
	Data           LazyCommentData `json:"data"`
}

type LazyCommentData struct {
	Cursor  LazyCommentCursor `json:"cursor"`
	Replies []CommentItem     `json:"replies"`
}

type LazyCommentCursor struct {
	IsBegin         bool            `json:"is_begin"`
	Prev            uint            `json:"prev"`
	Next            uint            `json:"next"`
	IsEnd           bool            `json:"is_end"`
	PaginationReply PaginationReply `json:"pagination_reply"`
}

type PaginationReply struct {
	NextOffset string `json:"next_offset"`
}

type CommentItem struct {
	Rpid         uint                `json:"rpid"`
	Member       CommentMember       `json:"member"`
	Oid          uint                `json:"oid"`
	Mid          uint                `json:"mid"`
	Content      CommentContent      `json:"content"`
	Replies      []CommentItem       `json:"replies"`
	ReplyControl CommentReplyControl `json:"reply_control"`
	Like         uint64              `json:"like"`
	Ctime        int64               `json:"ctime"`
}

type CommentContent struct {
	Message  string           `json:"message"`
	Pictures []CommentPicture `json:"pictures"`
	Emote    map[string]any   `json:"emote,omitempty"`
}

type CommentReplyControl struct {
	Location          string `json:"location"`
	SubReplyEntryText string `json:"sub_reply_entry_text"`
}

type CommentPicture struct {
	Src    string  `json:"img_src"`
	Width  int     `json:"img_width"`
	Height int     `json:"img_height"`
	Size   float64 `json:"img_size"`
}

type CommentMember struct {
	Uname  string `json:"uname"`
	Avatar string `json:"avatar"`
	Sex    string `json:"sex"`
}

type SubCommentResponse struct {
	ResponseCommon `json:"-"`
	Data           SubCommentData `json:"data"`
}

type SubCommentData struct {
	Page    SubCommentPage `json:"page"`
	Replies []CommentItem  `json:"replies"`
}

type SubCommentPage struct {
	Count int `json:"count"`
	Num   int `json:"num"`
}

// Search API responses

type SearchResponse struct {
	ResponseCommon `json:"-"`
	Data           SearchResponseData `json:"data"`
}

type SearchResponseData struct {
	Result []SearchItem `json:"result"`
}

type SearchItem struct {
	Typ         string `json:"type"`
	Id          uint   `json:"id"`
	Author      string `json:"author"`
	Pic         string `json:"pic"`
	Mid         uint   `json:"mid"`
	Title       string `json:"title"`
	Aid         uint   `json:"aid"`
	Bvid        string `json:"bvid"`
	Description string `json:"description"`
	Review      int    `json:"review"`
}

// Video info API response

type VideoInfoResponse struct {
	ResponseCommon `json:"-"`
	Data           VideoInfoData `json:"data"`
}

type VideoInfoData struct {
	Aid         uint           `json:"aid"`
	Bvid        string         `json:"bvid"`
	Title       string         `json:"title"`
	Pic         string         `json:"pic"`
	Description string         `json:"desc"`
	Owner       VideoInfoOwner `json:"owner"`
	Stat        VideoInfoStat  `json:"stat"`
}

type VideoInfoOwner struct {
	Mid  uint   `json:"mid"`
	Name string `json:"name"`
}

type VideoInfoStat struct {
	Reply uint `json:"reply"`
	Like  uint `json:"like"`
}

// User API response

type UserResponse struct {
	Code int `json:"code"`
	Data struct {
		Mid  uint   `json:"mid"`
		Name string `json:"name"`
		Face string `json:"face"`
		Sex  string `json:"sex"`
	} `json:"data"`
}
