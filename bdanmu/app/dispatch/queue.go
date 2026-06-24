package dispatch

import (
	"log"
	"sync/atomic"
	"time"

	"github.com/Yoak3n/libi/shared/domain/model/schema"
	"github.com/Yoak3n/libi/shared/domain/model/table"
	"github.com/Yoak3n/libi/shared/repository/implements"
)

const (
	danMuBatchSize    = 50
	entryBatchSize    = 50
	userBatchSize     = 10
	flushInterval     = 2 * time.Second
	danMuChanCapacity = 1000
	entryChanCapacity = 1000
	replyChanCapacity = 1000
)

type Queue struct {
	DanMu       chan *schema.DanMu
	UserEntry   chan *schema.UserEntry
	CollectUser chan uint
	reply       chan *schema.User

	danmuRepo *implements.DanMuRepository
	entryRepo *implements.UserEntryRepository
	emitFunc  func(event string, data ...any) bool
	getUsers  func(uids []uint) []*schema.User
}

func NewQueue(danmuRepo *implements.DanMuRepository, entryRepo *implements.UserEntryRepository, emitFunc func(string, ...any) bool, getUsers func([]uint) []*schema.User) *Queue {
	return &Queue{
		DanMu:       make(chan *schema.DanMu, danMuChanCapacity),
		UserEntry:   make(chan *schema.UserEntry, entryChanCapacity),
		CollectUser: make(chan uint, entryChanCapacity),
		reply:       make(chan *schema.User, replyChanCapacity),
		danmuRepo:   danmuRepo,
		entryRepo:   entryRepo,
		emitFunc:    emitFunc,
		getUsers:    getUsers,
	}
}

func (q *Queue) Start() {
	go q.collectDanMu()
	go q.collectUserEntry()
	go q.collectUserId()
	go q.emitLoop()
}

func (q *Queue) emitLoop() {
	for user := range q.reply {
		if q.emitFunc == nil {
			continue
		}
		msg := &Message{
			Type: MsgUser,
			Data: user,
		}
		q.emitFunc("message", msg)
	}
}

func (q *Queue) collectDanMu() {
	batch := make([]*table.DanMuTable, 0, danMuBatchSize)
	timer := time.NewTimer(flushInterval)
	var flag atomic.Int32

	go func() {
		for range timer.C {
			flag.Store(1)
			timer.Reset(flushInterval)
		}
	}()

	for d := range q.DanMu {
		batch = append(batch, &table.DanMuTable{
			MessageId: d.MessageId,
			Content:   d.Content,
			RoomId:    d.RoomId,
			Type:      d.Type,
			Sender:    d.Sender.UID,
		})
		if flag.Load() > 0 || len(batch) >= danMuBatchSize {
			flag.Store(0)
			go q.flushDanMu(batch)
			batch = make([]*table.DanMuTable, 0, danMuBatchSize)
		}
	}
}

func (q *Queue) collectUserEntry() {
	batch := make([]*table.UserEntryTable, 0, entryBatchSize)
	timer := time.NewTimer(flushInterval)
	var flag atomic.Int32

	go func() {
		for range timer.C {
			flag.Store(1)
			timer.Reset(flushInterval)
		}
	}()

	for entry := range q.UserEntry {
		batch = append(batch, &table.UserEntryTable{
			UID:       entry.UID,
			RoomId:    entry.RoomId,
			EnteredAt: entry.EnteredAt,
		})
		if flag.Load() > 0 || len(batch) >= entryBatchSize {
			flag.Store(0)
			go q.flushUserEntry(batch)
			batch = make([]*table.UserEntryTable, 0, entryBatchSize)
		}
	}
}

func (q *Queue) collectUserId() {
	pending := make([]uint, 0, userBatchSize)
	seen := make(map[uint]struct{}, userBatchSize)
	timer := time.NewTimer(flushInterval)
	var flag atomic.Int32

	go func() {
		for range timer.C {
			flag.Store(1)
			timer.Reset(flushInterval)
		}
	}()

	for uid := range q.CollectUser {
		if _, dup := seen[uid]; dup {
			continue
		}
		seen[uid] = struct{}{}
		pending = append(pending, uid)
		if flag.Load() > 0 || len(pending) >= userBatchSize {
			flag.Store(0)
			go q.processUserBatch(pending)
			pending = make([]uint, 0, userBatchSize)
			seen = make(map[uint]struct{}, userBatchSize)
		}
	}
}

func (q *Queue) processUserBatch(uids []uint) {
	if len(uids) == 0 || q.getUsers == nil {
		return
	}
	users := q.getUsers(uids)
	for _, u := range users {
		q.sendReply(u)
	}
}

func (q *Queue) flushUserEntry(records []*table.UserEntryTable) {
	if q.entryRepo == nil || len(records) == 0 {
		return
	}
	if err := q.entryRepo.CreateEntryBatch(records); err != nil {
		log.Printf("batch persist user entry error: %v", err)
	}
}

func (q *Queue) flushDanMu(records []*table.DanMuTable) {
	if q.danmuRepo == nil || len(records) == 0 {
		return
	}
	if err := q.danmuRepo.CreateDanMuBatch(records); err != nil {
		log.Printf("batch persist danmu error: %v", err)
	}
}

func (q *Queue) sendReply(user *schema.User) {
	select {
	case q.reply <- user:
	default:
	}
}
