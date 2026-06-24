package fetch

import (
	"errors"
	"fmt"

	"github.com/Yoak3n/libi/shared/config"
	"github.com/Yoak3n/libi/shared/domain/model/schema"
	"github.com/Yoak3n/libi/shared/package/request"
	"github.com/tidwall/gjson"
)

func GetRoomInfo(id int) (*schema.Room, error) {
	res, err := request.Get("https://api.live.bilibili.com/room/v1/Room/get_info", fmt.Sprintf("room_id=%d", config.Conf.RoomId))
	if err != nil {
		return nil, err
	}
	room := &schema.Room{
		ShortId: id,
		User:    &schema.User{},
	}
	result := gjson.ParseBytes(res)
	if result.Get("code").Int() == 0 {
		room.User.UID = uint(result.Get("data.uid").Int())
		room.LongId = result.Get("data.room_id").Int()
		room.FollowerCount = result.Get("data.attention").Int()
		room.Title = result.Get("data.title").String()
		room.Cover = result.Get("data.user_cover").String()
		user := GetUserInfo(room.User.UID)
		if user != nil {
			room.User = user
		}
		return room, nil
	}
	return nil, errors.New("get room information failed")

}
