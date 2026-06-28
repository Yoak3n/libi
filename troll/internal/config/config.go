package config

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/Yoak3n/libi/shared/config"
	"github.com/Yoak3n/libi/shared/login"
)

// CookieEntry pairs a cookie with its refresh token and UID.
type CookieEntry struct {
	UID          uint   `json:"uid,omitempty"`
	Cookie       string `json:"cookie"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

type TrollConfig struct {
	// Entries with refresh token (from login or DB)
	Entries []CookieEntry
	// Plain cookies without refresh token (from manual add)
	RawCookies []string
	Proxy      string
}

var Conf *TrollConfig

// loginClients holds login.Client instances, keyed by index in Entries.
var loginClients []*login.Client

func Init() {
	Conf = &TrollConfig{}
	if config.Conf == nil {
		return
	}
	Conf.Proxy = config.Conf.Proxy

	// Load all accounts from shared config
	if config.Conf.Auth != nil {
		for _, acc := range config.Conf.Auth.Accounts {
			if acc.RefreshToken != "" {
				uid := config.ExtractUID(acc.Cookie)
				Conf.Entries = append(Conf.Entries, CookieEntry{
					UID:          uid,
					Cookie:       acc.Cookie,
					RefreshToken: acc.RefreshToken,
				})
			} else {
				Conf.RawCookies = append(Conf.RawCookies, acc.Cookie)
			}
		}
	}
}

// AllCookies returns all valid cookie strings (entries + raw).
func (c *TrollConfig) AllCookies() []string {
	cookies := make([]string, 0, len(c.Entries)+len(c.RawCookies))
	for _, e := range c.Entries {
		if e.Cookie != "" {
			cookies = append(cookies, e.Cookie)
		}
	}
	cookies = append(cookies, c.RawCookies...)
	return cookies
}

// InitLoginClients creates login.Client instances for entries with refresh tokens.
// dbUpdateFunc is called when a cookie is refreshed, to persist to DB.
func InitLoginClients(dbUpdateFunc func(cookie, refreshToken string)) {
	loginClients = make([]*login.Client, len(Conf.Entries))
	for i := range Conf.Entries {
		idx := i
		entry := Conf.Entries[idx]
		loginClients[idx] = login.NewClient(entry.Cookie, entry.RefreshToken, func(cookie, refreshToken string) {
			Conf.Entries[idx].Cookie = cookie
			Conf.Entries[idx].RefreshToken = refreshToken
			// Update shared config's accounts by UID
			if config.Conf != nil && config.Conf.Auth != nil && entry.UID != 0 {
				if j := config.Conf.Auth.FindByUID(entry.UID); j >= 0 {
					config.Conf.Auth.Accounts[j].Cookie = cookie
					config.Conf.Auth.Accounts[j].RefreshToken = refreshToken
				}
			}
			if dbUpdateFunc != nil {
				dbUpdateFunc(cookie, refreshToken)
			}
		})
	}
}

// EnsureAllValid checks every entry, refreshes if needed.
func EnsureAllValid() {
	for i, client := range loginClients {
		if client == nil {
			continue
		}
		ok, err := client.IsLogin()
		if err != nil || ok {
			continue
		}
		if client.RefreshToken != "" {
			if err := client.RefreshCookie(); err == nil {
				continue
			}
		}
		Conf.Entries[i].Cookie = ""
	}
}

// GetLoginClient returns the login.Client at the given index, or nil.
func GetLoginClient(index int) *login.Client {
	if index < 0 || index >= len(loginClients) {
		return nil
	}
	return loginClients[index]
}

const CookieCheckUri = "https://passport.bilibili.com/x/passport-login/web/cookie/info"

type CookieInfoResponse struct {
	Code int `json:"code"`
}

// CheckCookie performs a basic validity check (no refresh).
func CheckCookie(cookie string) bool {
	req, _ := http.NewRequest("GET", CookieCheckUri, nil)
	req.Header.Set("Cookie", cookie)
	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return false
	}
	defer res.Body.Close()
	resBuf, err := io.ReadAll(res.Body)
	if err != nil {
		return false
	}
	response := &CookieInfoResponse{}
	err = json.Unmarshal(resBuf, response)
	return err == nil && response.Code == 0
}
