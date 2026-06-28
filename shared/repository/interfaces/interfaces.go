package interfaces

import (
	"time"

	"github.com/Yoak3n/libi/shared/domain/model/schema"
	"github.com/Yoak3n/libi/shared/domain/model/table"
)

type UserInterface interface {
	CreateUser(user *schema.User) error
	CreateOrUpdateUserBatch(users []*schema.User) error
	ReadUser(uid uint) (*schema.User, error)
	ReadUserBatchFresh(uids []uint, ttl time.Duration) (fresh []*schema.User, stale []uint, err error)
	UpdateUser(user *schema.User) error
	DeleteUser(uid uint) error
}

type DanMuInterface interface {
	CreateDanMuBatch(danmus []*table.DanMuTable) error
	ReadDanMuByRoom(roomId uint, limit int) ([]*table.DanMuTable, error)
	ReadDanMuByUser(uid uint, limit int) ([]*table.DanMuTable, error)
}

type VideoInterface interface {
	CreateVideo(video *table.VideoTable) error
	ReadVideo(avid uint) (*table.VideoTable, error)
	UpdateVideo(video *table.VideoTable) error
	DeleteVideo(avid uint) error
}

type LiveRoomInterface interface {
	CreateLiveRoom(room *table.LiveRoomTable) error
	ReadLiveRoom(roomId uint) (*table.LiveRoomTable, error)
	CreateOrUpdateLiveRoom(room *table.LiveRoomTable) error
	UpdateLiveRoom(room *table.LiveRoomTable) error
	DeleteLiveRoom(roomId uint) error
}

type MedalInterface interface {
	CreateMedal(medal *table.MedalTable) error
	ReadMedalsByUser(uid uint) ([]*table.MedalTable, error)
	UpdateMedal(medal *table.MedalTable) error
	DeleteMedal(id uint) error
}

type CommentInterface interface {
	CreateComment(comment *table.CommentTable) error
	CreateCommentBatch(comments []*table.CommentTable) error
	ReadComment(commentId uint) (*table.CommentTable, error)
	ReadCommentsByVideo(avid uint, limit int) ([]*table.CommentTable, error)
	UpdateComment(comment *table.CommentTable) error
	DeleteComment(commentId uint) error
}

type SignedUserInterface interface {
	CreateSignedUser(user *table.SignedUserTable) error
	ReadSignedUser(uid uint) (*table.SignedUserTable, error)
	ReadAllSignedUsers() ([]*table.SignedUserTable, error)
	UpdateSignedUser(user *table.SignedUserTable) error
	DeleteSignedUser(uid uint) error
}

type UserHistoryNameInterface interface {
	CreateHistoryName(name *table.UserHistoryNameTable) error
	ReadHistoryNames(uid uint) ([]*table.UserHistoryNameTable, error)
}

type UserEntryInterface interface {
	CreateEntry(entry *schema.UserEntry) error
	ReadEntriesByRoom(roomId uint, limit int) ([]*schema.UserEntry, error)
	ReadEntriesByUser(uid uint, limit int) ([]*schema.UserEntry, error)
	CountByUser(uid uint) (int64, error)
}

type ConfigurationInterface interface {
	CreateConfiguration(conf *table.ConfigurationTable) error
	ReadConfiguration(id uint) (*table.ConfigurationTable, error)
	ReadConfigurations() ([]*table.ConfigurationTable, error)
	ReadValidConfigurations() ([]*table.ConfigurationTable, error)
	ReadConfigurationByType(typ string) (*table.ConfigurationTable, error)
	UpdateConfiguration(conf *table.ConfigurationTable) error
	DeleteConfiguration(id uint) error
	DeleteConfigurations(ids []uint) error
}

// TrollQueryInterface provides troll-specific query methods using raw SQL
type TrollQueryInterface interface {
	QuerySimilarComments(topic string, n int) ([]schema.SimilarCommentGroup, error)
	QueryTopNUserInTopic(topic string, n int) ([]schema.UserQuery, error)
	QueryTopNUserInTopics(topics []string, n int) ([]schema.UserQuery, error)
	GetCommentsByVideo(avid uint) schema.CommentGroupByVideo
	SearchCommentWithKeyword(keyword string) []schema.CommentData
	GetVideosByTopic(topic string) []schema.VideoWithCommentCount
	GetAllTopicsList() []schema.TopicsData
	GetDashboardStats() schema.DashboardStats
	GetCommentsWithVideoFromUserInTopic(uid uint, topic string) []schema.CommentGroupByVideo
	UpdateTopic(topic string, newTopic string) error
	DeleteTopic(topic string) error
	UpdateTopicOfVideos(avid []uint, topic string) error
	DeleteVideos(avidList []uint) error
	GetSignedUserRecord() ([]*table.UserTable, error)
	GetSignedUserRecordByUID(uid uint) (*table.SignedUserTable, error)
}
