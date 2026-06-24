package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/Akegarasu/blivedm-go/client"
	"github.com/Akegarasu/blivedm-go/message"

	"github.com/Yoak3n/libi/shared/config"
	"github.com/Yoak3n/libi/shared/domain/model/schema"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"

	"bdanmu/app/dispatch"
)

const (
	EventStarted = "started"
	EventError   = "error"
	EventChange  = "change"
)

type LiveRoom struct {
	Emitter    EventEmitter
	Dispatcher *dispatch.Dispatcher
	cl         *client.Client
	running    bool
}

func (l *LiveRoom) ConnectRoom(id int) error {
	if id <= 0 {
		l.Emitter.Emit(EventError, "无效的直播间ID")
		return fmt.Errorf("invalid room id: %d", id)
	}
	if l.cl != nil && l.running {
		l.cl.Stop()
		l.cl = nil
		l.running = false
	}
	config.Conf.RoomId = id
	l.cl = client.NewClient(id)
	l.cl.SetCookie(config.Conf.Auth.Cookie)
	l.registerHandler()

	if err := l.cl.Start(); err != nil {
		l.Emitter.Emit(EventError, fmt.Sprintf("连接直播间失败: %v", err))
		return err
	}
	l.running = true

	roomInfo := fmt.Sprintf(`{"short_id":%d,"title":"直播间 %d"}`, id, id)
	l.Emitter.Emit(EventStarted, roomInfo)
	return nil
}

func (l *LiveRoom) registerHandler() {
	l.cl.OnDanmaku(l.messageHandler)
	l.cl.RegisterCustomEventHandler("INTERACT_WORD", l.userEntryHandler)
	l.cl.OnSuperChat(l.superChatHandler)
}

func (l *LiveRoom) messageHandler(msg *message.Danmaku) {
	if msg.Type == message.EmoticonDanmaku {
		msg.Content = fmt.Sprintf("<img src='%s' max-width='100px' />", msg.Emoticon.Url)
	} else {
		result := gjson.Get(msg.Raw, "info.0.15.extra").String()
		if emots := gjson.Get(result, "emots"); emots.Exists() {
			for k, emot := range emots.Map() {
				width := emot.Get("width").String()
				height := emot.Get("height").String()
				src := emot.Get("url").String()
				msg.Content = strings.ReplaceAll(msg.Content, k, fmt.Sprintf("<img width='%s' src='%s' height='%s' />", width, src, height))
			}
		}
	}
	uid := uint(msg.Sender.Uid)
	user := &schema.User{
		UID:   uid,
		Name:  msg.Sender.Uname,
		Guard: msg.Sender.GuardLevel > 0,
	}
	if msg.Sender.Medal != nil {
		user.Medal = &schema.Medal{
			Name:     msg.Sender.Medal.Name,
			Level:    msg.Sender.Medal.Level,
			Color:    msg.Sender.Medal.Color,
			TargetID: uint(msg.Sender.Medal.UpUid),
			OwnerID:  uid,
		}
	}

	danMu := &schema.DanMu{
		Content:   msg.Content,
		Sender:    *user,
		MessageId: uuid.NewString(),
		RoomId:    uint(config.Conf.RoomId),
	}
	l.Dispatcher.Dispatch(&dispatch.Message{
		Type: dispatch.MsgDanMu,
		Data: danMu,
	})
	l.Dispatcher.CollectUser(uid)
}

func (l *LiveRoom) userEntryHandler(s string) {
	result := gjson.Parse(s)
	data := result.Get("data")
	uid := uint(data.Get("uid").Int())
	entry := &schema.UserEntry{
		UID:       uid,
		Name:      data.Get("uname").String(),
		RoomId:    uint(data.Get("roomid").Int()),
		EnteredAt: time.Now(),
	}
	l.Dispatcher.Dispatch(&dispatch.Message{
		Type: dispatch.MsgUserEntry,
		Data: entry,
	})
}

func (l *LiveRoom) superChatHandler(s *message.SuperChat) {
	superChat := &schema.SuperChat{
		User: &schema.User{
			UID:    uint(s.Uid),
			Name:   s.UserInfo.Uname,
			Avatar: s.UserInfo.Face,
			Guard:  s.UserInfo.GuardLevel > 0,
			Medal: &schema.Medal{
				Name:     s.MedalInfo.MedalName,
				OwnerID:  uint(s.Uid),
				Level:    s.MedalInfo.MedalLevel,
				TargetID: uint(s.MedalInfo.TargetId),
			},
		},
		RoomID:    config.Conf.RoomId,
		MessageID: uuid.NewString(),
		Content:   s.Message,
		Timestamp: s.Ts,
		EndTime:   s.EndTime,
		Price:     s.Price,
	}
	l.Dispatcher.Dispatch(&dispatch.Message{
		Type: dispatch.MsgSuperChat,
		Data: superChat,
	})
}
