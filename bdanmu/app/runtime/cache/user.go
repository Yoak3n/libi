package cache

import (
	"bdanmu/app/runtime/fetch"
	"container/list"
	"sync"
	"time"

	"github.com/Yoak3n/libi/shared/config"
	"github.com/Yoak3n/libi/shared/domain/model/schema"
	"github.com/Yoak3n/libi/shared/repository/implements"
	"gorm.io/gorm"
)

var userRepo *implements.UserRepository

const maxCacheSize = 5000

type cacheEntry struct {
	uid  uint
	user *schema.User
}

type lruCache struct {
	mu       sync.Mutex
	items    map[uint]*list.Element
	order    *list.List // front = newest, back = oldest
}

func newLRUCache() *lruCache {
	return &lruCache{
		items: make(map[uint]*list.Element, maxCacheSize),
		order: list.New(),
	}
}

func (c *lruCache) Get(uid uint) (*schema.User, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[uid]; ok {
		c.order.MoveToFront(el)
		return el.Value.(*cacheEntry).user, true
	}
	return nil, false
}

func (c *lruCache) Put(uid uint, user *schema.User) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[uid]; ok {
		el.Value.(*cacheEntry).user = user
		c.order.MoveToFront(el)
		return
	}
	if c.order.Len() >= maxCacheSize {
		back := c.order.Back()
		if back != nil {
			evicted := c.order.Remove(back).(*cacheEntry)
			delete(c.items, evicted.uid)
		}
	}
	entry := &cacheEntry{uid: uid, user: user}
	el := c.order.PushFront(entry)
	c.items[uid] = el
}

var memCache = newLRUCache()

func Init(db *gorm.DB) {
	userRepo = implements.NewUserRepository(db)
}

func cacheTTL() time.Duration {
	return time.Duration(config.Conf.CacheTTL) * time.Hour
}

func GetUserInfo(uid uint) *schema.User {
	ttl := cacheTTL()

	// 1. In-memory LRU cache
	if u, ok := memCache.Get(uid); ok {
		return u
	}

	// 2. Database
	if userRepo != nil {
		fresh, _, err := userRepo.ReadUserBatchFresh([]uint{uid}, ttl)
		if err == nil && len(fresh) > 0 {
			u := fresh[0]
			memCache.Put(uid, u)
			return u
		}
	}

	// 3. Bilibili API
	user := fetch.GetUserInfo(uid)
	if user != nil {
		memCache.Put(uid, user)
		if userRepo != nil {
			_ = userRepo.CreateOrUpdateUserBatch([]*schema.User{user})
		}
	}
	return user
}

func GetUserInfoMultiply(uids []uint) []*schema.User {
	ttl := cacheTTL()
	var result []*schema.User
	var toFetch []uint

	// 1. In-memory LRU cache
	for _, uid := range uids {
		if u, ok := memCache.Get(uid); ok {
			result = append(result, u)
		} else {
			toFetch = append(toFetch, uid)
		}
	}

	if len(toFetch) == 0 {
		return result
	}

	// 2. Database
	if userRepo != nil {
		fresh, stale, err := userRepo.ReadUserBatchFresh(toFetch, ttl)
		if err == nil {
			for _, u := range fresh {
				memCache.Put(u.UID, u)
				result = append(result, u)
			}
			toFetch = stale
		}
	}

	// 3. Bilibili API
	if len(toFetch) > 0 {
		fetched := fetch.GetUserInfoMultiply(toFetch)
		if fetched != nil {
			for _, u := range fetched {
				memCache.Put(u.UID, u)
			}
			result = append(result, fetched...)
			if userRepo != nil {
				_ = userRepo.CreateOrUpdateUserBatch(fetched)
			}
		}
	}
	return result
}
