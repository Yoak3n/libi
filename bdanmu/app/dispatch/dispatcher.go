package dispatch

import (
	"log"

	"github.com/Yoak3n/libi/shared/domain/model/schema"
	"github.com/Yoak3n/libi/shared/repository/implements"
	"gorm.io/gorm"
)

type EventEmitter interface {
	Emit(event string, data ...any) bool
}

type Dispatcher struct {
	emitter EventEmitter
	ws      *WSServer
	queue   *Queue
}

func NewDispatcher(emitter EventEmitter, db *gorm.DB, getUsers func([]uint, map[uint]*schema.Medal) []*schema.User) *Dispatcher {
	d := &Dispatcher{
		emitter: emitter,
		ws:      NewWSServer(),
	}
	if db != nil {
		danmuRepo := implements.NewDanMuRepository(db)
		entryRepo := implements.NewUserEntryRepository(db)
		d.queue = NewQueue(danmuRepo, entryRepo, emitter.Emit, getUsers)
		d.queue.Start()
	}
	return d
}

func (d *Dispatcher) StartWS(addr string) {
	go func() {
		if err := d.ws.Start(addr); err != nil {
			log.Printf("WS server error: %v", err)
		}
	}()
}

func (d *Dispatcher) StopWS() {
	d.ws.Stop()
}

func (d *Dispatcher) IsWSRunning() bool {
	return d.ws.IsRunning()
}

func (d *Dispatcher) Dispatch(msg *Message) {
	// 1. Wails 事件 → 前端
	d.emitter.Emit("message", msg)

	// 2. WS 广播 → 外部客户端
	d.ws.Broadcast(msg)

	// 3. 入队，批量写入数据库
	d.enqueue(msg)
}

func (d *Dispatcher) CollectUser(user *schema.User) {
	if d.queue == nil {
		return
	}
	cu := collectedUser{uid: user.UID}
	if user.Medal != nil {
		cu.medal = user.Medal
	}
	select {
	case d.queue.CollectUser <- cu:
	default:
		log.Printf("[dispatch] collectUser chan full, dropping uid=%d", user.UID)
	}
}

func (d *Dispatcher) enqueue(msg *Message) {
	if d.queue == nil {
		return
	}
	switch msg.Type {
	case MsgDanMu:
		if dm, ok := msg.Data.(*schema.DanMu); ok {
			select {
			case d.queue.DanMu <- dm:
			default:
			}
		}
	case MsgUserEntry:
		if entry, ok := msg.Data.(*schema.UserEntry); ok {
			select {
			case d.queue.UserEntry <- entry:
			default:
			}
		}
	}
}
