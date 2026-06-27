package tui

import (
	"time"

	"github.com/Yoak3n/libi/shared/domain/model/schema"
)

type viewState int

const (
	viewMenu viewState = iota
	viewTopics
	viewVideos
	viewComments
	viewSearch
	viewSearchResults
	viewDashboard
	viewTopUsers
	viewUserComments
	viewSimilar
	viewTopicSelect
	viewSignedUsers
	viewAddSignedUser
	viewEditSignedUser
)

type sortMode int

const (
	sortByHot sortMode = iota
	sortByTime
	sortByTimeOld
)

var menuItems = []struct {
	label       string
	description string
}{
	{"Browse", "Explore topics, videos, and comments"},
	{"Search", "Search comments by keyword"},
	{"Top Users", "Top commenters per topic"},
	{"Similar Comments", "Repeated/copied comments per topic"},
	{"Marked Users", "Manage marked users and notes"},
	{"Dashboard", "Overview statistics"},
}

type App struct {
	state  viewState
	topics []topicItem
	videos      []videoItem
	comments    []commentItem
	cursor      int
	topicName   string
	searchInput string
	err         error
	width       int
	height      int
	quitting    bool

	menuCursor      int
	topUsers        []topUserItem
	similarComments []similarItem
	stats           schema.DashboardStats
	searchResults   []commentItem
	topicSelectFor  string // "topUsers" or "similar"
	videoTitle      string
	flatComments    []commentItem // flattened comments (root + children) for display

	selectedUser    string
	selectedUID     uint
	userComments    []userVideoGroup
	flatUserItems   []userCommentDisplayItem

	commentSortMode sortMode
	scrollToTop     bool // force scroll to top on next render

	signedUsers      []signedUserItem
	signedUserUID    string // UID input buffer for add
	signedUserInput  string // description input buffer for add/edit
	signedUserField  string // "uid" or "description" during add/edit
	editTargetUID    uint   // UID being edited
}

type topicItem struct {
	Name  string
	Count int64
}

type videoItem struct {
	Avid     uint
	Bvid     string
	Title    string
	Count    int
	UpdateAt string
}

type commentItem struct {
	Id        uint
	Content   string
	Owner     string
	OwnerUID  uint
	Like      uint64
	IsChild   bool // true for sub-comments rendered with └
	CreatedAt time.Time
	Children  []commentItem
}

type topUserItem struct {
	Rank     int
	Username string
	UID      uint
	Count    int
}

type similarItem struct {
	Text  string
	Count int
	Ids   string
}

type userVideoGroup struct {
	VideoTitle string
	Bvid       string
	Comments   []commentItem
}

type signedUserItem struct {
	UID         uint
	Username    string
	Description string
	LastViewed  time.Time
}

type userCommentDisplayItem struct {
	IsHeader   bool
	VideoTitle string
	Bvid       string
	Content    string
	CommentId  uint
	CreatedAt  time.Time
}

type displayLine struct {
	text    string
	itemIdx int
	isChild bool
}

// --- Message types ---

type topicsLoadedMsg []topicItem
type videosLoadedMsg []videoItem
type commentsLoadedMsg struct {
	items []commentItem
	title string
}
type searchResultsLoadedMsg []commentItem
type topUsersLoadedMsg []topUserItem
type similarLoadedMsg []similarItem
type dashboardLoadedMsg schema.DashboardStats
type userCommentsLoadedMsg []userVideoGroup
type signedUsersLoadedMsg []signedUserItem
type errMsg error
