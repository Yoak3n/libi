package config

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Yoak3n/libi/shared/login"
	"github.com/spf13/viper"
	"go.yaml.in/yaml/v3"
)

type (
	Configuration struct {
		RoomId    int       `yaml:"room_id"`
		Proxy     string    `yaml:"proxy"`
		Extension bool      `yaml:"extension"`
		Auth      *Auth     `yaml:"auth"`
		Database  *Database `yaml:"database"`
		CacheTTL  int       `yaml:"cache_ttl_hours"`
	}
	Auth struct {
		Accounts   []AccountEntry `yaml:"accounts"`
		ImgKey     string         `yaml:"img_key"`
		SubKey     string         `yaml:"sub_key"`
		LastUpdate int64          `yaml:"last_update"`
	}
	AccountEntry struct {
		UID          uint   `yaml:"uid,omitempty"`
		Cookie       string `yaml:"cookie"`
		RefreshToken string `yaml:"refresh_token,omitempty"`
	}
	Database struct {
		Type     string `yaml:"type"`
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		Name     string `yaml:"database"`
	}
)

var (
	Conf *Configuration
	v    *viper.Viper
)

func init() {
	_, err1 := os.Stat("config.yaml")
	_, err2 := os.Stat("config.yml")
	if os.IsNotExist(err1) && os.IsNotExist(err2) {
		fp, _ := os.Create("config.yaml")
		defer fp.Close()
	}
	v = viper.New()
	Conf = &Configuration{
		Auth:     &Auth{},
		Database: &Database{},
	}
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	err := v.ReadInConfig()
	if err != nil {
		return
	}
	v.SetDefault("database.type", "sqlite")
	v.SetDefault("database.name", "bliveDB")
	v.SetDefault("cache_ttl_hours", 24)
	loadConfig()
	v.WatchConfig()
}

// Reload forces a re-read of the config file.
func Reload() {
	if v == nil {
		return
	}
	_ = v.ReadInConfig()
	loadConfig()
}

// loadConfig reads flat values via viper, and auth accounts via direct YAML parse.
// Viper normalizes keys (lowercases, strips underscores) which breaks nested struct
// deserialization for fields like refresh_token. So we always parse auth from YAML.
func loadConfig() {
	_ = v.Unmarshal(Conf)
	loadAccountsFromYAML()
}

// loadAccountsFromYAML reads the auth section directly from the config file.
func loadAccountsFromYAML() {
	configFile := v.ConfigFileUsed()
	if configFile == "" {
		return
	}
	data, err := os.ReadFile(configFile)
	if err != nil {
		return
	}
	var raw struct {
		Auth struct {
			Accounts []AccountEntry `yaml:"accounts"`
			// Legacy single-cookie format
			Cookie      string `yaml:"cookie"`
			RefreshToken string `yaml:"refresh_token"`
		} `yaml:"auth"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return
	}
	if len(raw.Auth.Accounts) > 0 {
		Conf.Auth.Accounts = raw.Auth.Accounts
	} else if raw.Auth.Cookie != "" {
		Conf.Auth.Accounts = []AccountEntry{
			{UID: ExtractUID(raw.Auth.Cookie), Cookie: raw.Auth.Cookie, RefreshToken: raw.Auth.RefreshToken},
		}
	}
}

// PrimaryCookie returns the first account's cookie, or empty string.
func (a *Auth) PrimaryCookie() string {
	if len(a.Accounts) > 0 {
		return a.Accounts[0].Cookie
	}
	return ""
}

// PrimaryRefreshToken returns the first account's refresh token, or empty string.
func (a *Auth) PrimaryRefreshToken() string {
	if len(a.Accounts) > 0 {
		return a.Accounts[0].RefreshToken
	}
	return ""
}

// SetPrimary sets or adds the first account.
func (a *Auth) SetPrimary(cookie, refreshToken string) {
	if len(a.Accounts) == 0 {
		a.Accounts = append(a.Accounts, AccountEntry{})
	}
	a.Accounts[0].Cookie = cookie
	a.Accounts[0].RefreshToken = refreshToken
}

// AddAccount appends a new account. UID is extracted from cookie if not provided.
func (a *Auth) AddAccount(cookie, refreshToken string) {
	entry := AccountEntry{
		Cookie:       cookie,
		RefreshToken: refreshToken,
	}
	entry.UID = ExtractUID(cookie)
	a.Accounts = append(a.Accounts, entry)
}

// FindByUID returns the index of the account with the given UID, or -1.
func (a *Auth) FindByUID(uid uint) int {
	for i, acc := range a.Accounts {
		if acc.UID == uid {
			return i
		}
	}
	return -1
}

// ExtractUID extracts DedeUserID from a bilibili cookie string.
func ExtractUID(cookie string) uint {
	for _, part := range strings.Split(cookie, ";") {
		part = strings.TrimSpace(part)
		if key, val, ok := strings.Cut(part, "="); ok && key == "DedeUserID" {
			if uid, err := strconv.ParseUint(val, 10, 64); err == nil {
				return uint(uid)
			}
		}
	}
	return 0
}

// RemoveAccount removes the account at the given index.
func (a *Auth) RemoveAccount(index int) {
	if index < 0 || index >= len(a.Accounts) {
		return
	}
	a.Accounts = append(a.Accounts[:index], a.Accounts[index+1:]...)
}

// EnsureAccounts validates all accounts. Invalid cookies are refreshed
// if a refresh token is available. Accounts that can't be recovered are removed.
// Returns the number of removed accounts.
func (a *Auth) EnsureAccounts() int {
	removed := 0
	for i := 0; i < len(a.Accounts); {
		entry := a.Accounts[i]
		if isCookieValid(entry.Cookie) {
			i++
			continue
		}
		// Cookie invalid — try refresh
		if entry.RefreshToken != "" {
			client := login.NewClient(entry.Cookie, entry.RefreshToken, func(cookie, refreshToken string) {
				a.Accounts[i].Cookie = cookie
				a.Accounts[i].RefreshToken = refreshToken
			})
			if err := client.RefreshCookie(); err == nil {
				i++
				continue
			}
		}
		// Can't recover — remove
		a.Accounts = append(a.Accounts[:i], a.Accounts[i+1:]...)
		removed++
	}
	if removed > 0 {
		SaveAuth()
	}
	return removed
}

func isCookieValid(cookie string) bool {
	req, _ := http.NewRequest("GET", "https://passport.bilibili.com/x/passport-login/web/cookie/info", nil)
	req.Header.Set("Cookie", cookie)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func SetWBIKey(img string, sub string) {
	v.Set("auth.img_key", img)
	v.Set("auth.sub_key", sub)
	Conf.Auth.ImgKey = img
	Conf.Auth.SubKey = sub
	t := time.Now().Unix()
	Conf.Auth.LastUpdate = t
	v.Set("auth.last_update", t)
	err := v.WriteConfig()
	if err != nil {
		return
	}
}

// SaveAuth writes the entire auth section to config.yaml using direct YAML marshal,
// bypassing viper to preserve field names like refresh_token.
func SaveAuth() {
	configFile := v.ConfigFileUsed()
	if configFile == "" {
		return
	}

	// Read existing file to preserve non-auth fields
	data, _ := os.ReadFile(configFile)
	var raw map[string]interface{}
	_ = yaml.Unmarshal(data, &raw)

	// Build auth section
	authMap := map[string]interface{}{
		"accounts": Conf.Auth.Accounts,
	}
	raw["auth"] = authMap

	out, err := yaml.Marshal(raw)
	if err != nil {
		return
	}
	_ = os.WriteFile(configFile, out, 0644)

	// Sync viper's internal state for non-auth fields
	_ = v.ReadInConfig()
}

func SetRoomId(id int) error {
	v.Set("room_id", id)
	Conf.RoomId = id
	err := v.WriteConfig()
	if err != nil {
		return err
	}
	return nil
}
