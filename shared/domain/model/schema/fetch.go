package schema

import (
	"time"

	"github.com/Yoak3n/libi/shared/domain/model/table"
)

type User struct {
	UID           uint   `json:"uid"`
	Name          string `json:"name"`
	Sex           int    `json:"sex"`
	Avatar        string `json:"avatar"`
	Guard         bool   `json:"guard"`
	FollowerCount int64  `json:"fans_count,omitempty"`
	Medal         *Medal `json:"medal,omitempty"`
}

type DanMu struct {
	Sender    User   `json:"user"`
	Content   string `json:"content"`
	RoomId    uint   `json:"room_id"`
	Type      int8   `json:"type"`
	MessageId string `json:"message_id"`
}

type Medal struct {
	Name     string `json:"name"`
	OwnerID  uint   `json:"owner_id"`
	Level    int    `json:"level,omitempty"`
	TargetID uint   `json:"target_id"`
	Color    int    `json:"color,omitempty"`
}

type Room struct {
	*User         `json:"user"`
	Title         string `json:"title"`
	Cover         string `json:"cover"`
	ShortId       int    `json:"short_id"`
	LongId        int64  `json:"long_id"`
	FollowerCount int64  `json:"follower_count"`
}

type UserEntry struct {
	UID       uint      `json:"uid"`
	Name      string    `json:"name"`
	RoomId    uint      `json:"room_id"`
	EnteredAt time.Time `json:"entered_at"`
}

type SuperChat struct {
	User      *User  `json:"user"`
	Content   string `json:"content"`
	RoomID    int    `json:"room_id"`
	MessageID string `json:"message_id"`
	Timestamp int    `json:"timestamp"`
	EndTime   int    `json:"end_time"`
	Price     int    `json:"price"`
}

func ToModel(u *table.UserTable) *User {
	user := &User{
		UID:           u.UID,
		Name:          u.Name,
		Sex:           u.Sex,
		Avatar:        u.Avatar,
		Guard:         u.Guard,
		FollowerCount: u.FollowerCount,
	}
	if len(u.Medals) > 0 {
		m := u.Medals[0]
		user.Medal = &Medal{
			Name:     m.Name,
			OwnerID:  m.Owner,
			Level:    m.Level,
			TargetID: m.Target,
			Color:    m.Color,
		}
	}
	return user
}
