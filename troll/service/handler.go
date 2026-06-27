package service

import (
	"strings"
	"time"

	sharedconfig "github.com/Yoak3n/libi/shared/config"
	"github.com/Yoak3n/libi/shared/database"
	"github.com/Yoak3n/libi/shared/repository/implements"
	"github.com/Yoak3n/libi/shared/repository/interfaces"
	trollconfig "troll/internal/config"
	"troll/internal/limiter"
)

var (
	UserRepo       interfaces.UserInterface
	VideoRepo      interfaces.VideoInterface
	CommentRepo    interfaces.CommentInterface
	TrollRepo      interfaces.TrollQueryInterface
	ConfRepo       interfaces.ConfigurationInterface
	SignedUserRepo interfaces.SignedUserInterface
	accountLimiter *limiter.AccountLimiter
	dispatcher     *limiter.Dispatcher
)

func Init(requestInterval time.Duration) {
	trollconfig.Init()

	accountLimiter = limiter.NewAccountLimiter(requestInterval)

	// Set up the refresh callback — finds the entry by UID, refreshes via shared/login
	accountLimiter.SetRefreshFunc(func(id uint, oldCookie string) string {
		oldUID := sharedconfig.ExtractUID(oldCookie)
		for i, entry := range trollconfig.Conf.Entries {
			if entry.UID == oldUID || entry.Cookie == oldCookie {
				client := trollconfig.GetLoginClient(i)
				if client != nil {
					if err := client.RefreshCookie(); err == nil {
						return client.Cookie
					}
				}
				break
			}
		}
		return ""
	})

	// Register all cookies with the limiter
	cookies := trollconfig.Conf.AllCookies()
	for i, cookie := range cookies {
	 refreshToken := ""
		if i < len(trollconfig.Conf.Entries) {
			refreshToken = trollconfig.Conf.Entries[i].RefreshToken
		}
		accountLimiter.SetAccount(uint(i+1), cookie, refreshToken)
	}

	// Initialize login clients — on refresh, persist to config.yaml
	trollconfig.InitLoginClients(func(cookie, refreshToken string) {
		// shared/config is already updated by InitLoginClients callback
		// Just persist to file
		if sharedconfig.Conf != nil {
			sharedconfig.SaveAuth()
		}
	})

	// Start the centralized request dispatcher
	dispatcher = limiter.NewDispatcher(accountLimiter, 64)
	dispatcher.Start(1)
}

func InitDB() {
	database.InitDatabase()
	db := database.DB()
	UserRepo = implements.NewUserRepository(db)
	VideoRepo = implements.NewVideoRepository(db)
	CommentRepo = implements.NewCommentRepository(db)
	TrollRepo = implements.NewTrollQueryRepository(db)
	ConfRepo = implements.NewConfigurationRepository(db)
	SignedUserRepo = implements.NewSignedUserRepository(db)
}

type Handler struct {
	Title string
	Topic string
	Cache string
	Bvid  string
	Avid  int64
}

func NewHandler(cache, title, topic, bvid string, avid int64) *Handler {
	return &Handler{
		Title: title,
		Topic: topic,
		Cache: cache,
		Bvid:  bvid,
		Avid:  avid,
	}
}

func (h *Handler) Run() {
	if h.Topic != "" {
		NewTopic(h.Cache, h.Title, strings.Split(h.Topic, ","))
		return
	}
	if h.Bvid != "" || h.Avid != -1 {
		NewVideo(h.Cache, h.Title, h.Bvid, h.Avid)
		return
	}
}

func GetAccountLimiter() *limiter.AccountLimiter {
	return accountLimiter
}

func GetDispatcher() *limiter.Dispatcher {
	return dispatcher
}

func GetProxy() string {
	if sharedconfig.Conf != nil {
		return sharedconfig.Conf.Proxy
	}
	return ""
}

// EnsureDB ensures database and repos are initialized
func EnsureDB() {
	if database.DB() == nil {
		InitDB()
	}
	if TrollRepo == nil {
		db := database.DB()
		UserRepo = implements.NewUserRepository(db)
		VideoRepo = implements.NewVideoRepository(db)
		CommentRepo = implements.NewCommentRepository(db)
		TrollRepo = implements.NewTrollQueryRepository(db)
		ConfRepo = implements.NewConfigurationRepository(db)
		SignedUserRepo = implements.NewSignedUserRepository(db)
	}
}
