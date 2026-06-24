package service

import (
	"fmt"
	"log"
	"time"

	"bdanmu/app/dispatch"
	"github.com/Yoak3n/libi/shared/config"
	"github.com/Yoak3n/libi/shared/login"
)

const (
	EventQRCode      = "auth:qr-url"
	EventLoginResult = "auth:login-result"
	LoginTimeout     = 3 * time.Minute
)

type EventEmitter interface {
	Emit(event string, data ...any) bool
}

type AuthService struct {
	Emitter    EventEmitter
	Dispatcher *dispatch.Dispatcher
}

// CheckLogin checks if the current cookie is valid (refreshes if needed).
// Returns true if logged in. Returns false and emits auth:need-login if not.
func (a *AuthService) CheckLogin() bool {
	client := a.newClient()

	// Try existing cookie
	ok, err := client.IsLogin()
	if err == nil && ok {
		return true
	}

	// Try refreshing cookie
	if config.Conf.Auth.RefreshToken != "" {
		if err := client.RefreshCookie(); err == nil {
			ok, err = client.IsLogin()
			if err == nil && ok {
				return true
			}
		}
	}

	return false
}

// StartLogin generates a QR code and starts polling for login.
// Called by the frontend after it has navigated to the login page.
func (a *AuthService) StartLogin() {
	url, key, err := login.GenerateQRCode()
	if err != nil {
		log.Printf("generate QR code failed: %v", err)
		a.Emitter.Emit(EventLoginResult, false)
		return
	}

	a.Emitter.Emit(EventQRCode, url)

	go func() {
		client := a.newClient()
		done := make(chan error, 1)
		go func() {
			done <- client.PollLogin(key)
		}()

		select {
		case err := <-done:
			if err != nil {
				a.Emitter.Emit(EventLoginResult, false)
			} else {
				a.Emitter.Emit(EventLoginResult, true)
			}
		case <-time.After(LoginTimeout):
			a.Emitter.Emit(EventLoginResult, false)
		}
	}()
}

// SyncAuth returns the current cookie and refresh token from config.
func (a *AuthService) SyncAuth() [2]string {
	return [2]string{config.Conf.Auth.Cookie, config.Conf.Auth.RefreshToken}
}

// SyncRoomId returns the current room ID from config.
func (a *AuthService) SyncRoomId() int {
	return config.Conf.RoomId
}

func (a *AuthService) StartWS(port int) {
	if a.Dispatcher != nil {
		a.Dispatcher.StartWS(fmt.Sprintf(":%d", port))
	}
}

func (a *AuthService) StopWS() {
	if a.Dispatcher != nil {
		a.Dispatcher.StopWS()
	}
}

func (a *AuthService) IsWSRunning() bool {
	if a.Dispatcher != nil {
		return a.Dispatcher.IsWSRunning()
	}
	return false
}

func (a *AuthService) newClient() *login.Client {
	return login.NewClient(config.Conf.Auth.Cookie, config.Conf.Auth.RefreshToken, func(cookie, refreshToken string) {
		config.Conf.Auth.Cookie = cookie
		config.Conf.Auth.RefreshToken = refreshToken
		config.SaveAuth()
	})
}
