package table

import (
	"time"

	"gorm.io/gorm"
)

type VideoTable struct {
	Avid        uint   `json:"avid" gorm:"primaryKey"`
	Bvid        string `json:"bvid"`
	Title       string `json:"title"`
	Cover       string
	Topic       string
	Description string `json:"description"`
	Owner       uint   `json:"owner" gorm:"index"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

type LiveRoomTable struct {
	RoomId        uint `gorm:"primaryKey"`
	Owner         uint `gorm:"index"`
	ShortId       uint
	Title         string
	Cover         string
	LongId        int64
	FollowerCount int64
	DanMuList     []DanMuTable `gorm:"foreignKey:RoomId;references:RoomId"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}

type UserHistoryNameTable struct {
	gorm.Model
	UID  uint `gorm:"index"`
	Name string
}

type UserTable struct {
	UID           uint `gorm:"primaryKey"`
	Name          string
	Sex           int
	Avatar        string
	Guard         bool
	FollowerCount int64
	HistoryNames  []UserHistoryNameTable `gorm:"foreignKey:UID;references:UID"`
	Medals        []MedalTable           `gorm:"foreignKey:Owner;references:UID"`
	DanMuList     []DanMuTable           `gorm:"foreignKey:Sender;references:UID"`
	Videos        []VideoTable           `gorm:"foreignKey:Owner;references:UID"`
	Comments      []CommentTable         `gorm:"foreignKey:Owner;references:UID"`
	LiveRoom      LiveRoomTable          `gorm:"foreignKey:Owner;references:UID"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}

type SignedUserTable struct {
	UID         uint `gorm:"primaryKey"`
	Description string
	LastViewed  time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

type MedalTable struct {
	gorm.Model
	Owner  uint `gorm:"uniqueIndex:idx_owner_target"`
	Name   string
	Level  int
	Target uint `gorm:"uniqueIndex:idx_owner_target"`
	Color  int
}

type DanMuTable struct {
	MessageId string `gorm:"primaryKey"`
	Content   string
	RoomId    uint  `gorm:"index:idx_room_created,priority:1"`
	Type      int8
	Sender    uint `gorm:"index:idx_sender_created,priority:1"`
	CreatedAt time.Time `gorm:"index:idx_room_created,priority:2;index:idx_sender_created,priority:2"`
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type UserEntryTable struct {
	ID        uint `gorm:"primaryKey;autoIncrement"`
	UID       uint `gorm:"index:idx_uid_entered,priority:1"`
	RoomId    uint `gorm:"index:idx_room_entered,priority:1"`
	EnteredAt time.Time `gorm:"index:idx_uid_entered,priority:2;index:idx_room_entered,priority:2"`
}

type ConfigurationTable struct {
	Type         string
	Data         string
	RefreshToken string `gorm:"column:refresh_token"`
	Invalid      bool   `gorm:"default:false"`
	gorm.Model
}

type CommentTable struct {
	Text          string `json:"text"`
	Owner         uint   `gorm:"index"`
	VideoAvid     uint   `gorm:"index:idx_avid_created,priority:1"`
	ParentComment uint
	CommentId     uint   `gorm:"primaryKey"`
	Like          uint64
	CommentTime   time.Time      `gorm:"column:comment_time;index:idx_avid_created,priority:2"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}
