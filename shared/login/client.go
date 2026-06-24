package login

import (
	"errors"
	"io"
	"net/http"

	"github.com/tidwall/gjson"
)

const userAgent = `Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/97.0.4692.99 Safari/537.36 Edg/97.0.1072.69`

// Client holds authentication state for Bilibili API requests.
type Client struct {
	Cookie       string
	RefreshToken string
	onUpdate     func(cookie, refreshToken string)
}

// NewClient creates a login client. onUpdate is called when cookie or refresh token changes.
func NewClient(cookie, refreshToken string, onUpdate func(cookie, refreshToken string)) *Client {
	return &Client{
		Cookie:       cookie,
		RefreshToken: refreshToken,
		onUpdate:     onUpdate,
	}
}

func (c *Client) save() {
	if c.onUpdate != nil {
		c.onUpdate(c.Cookie, c.RefreshToken)
	}
}

// EnsureLogin checks if the current cookie is valid, tries to refresh it, and falls back to QR code login.
// Returns nil if logged in (cookie is valid). Returns the QR code URL and key via callback if QR login is needed.
// If onQRCode is nil and cookie is invalid, returns an error.
func (c *Client) EnsureLogin(onQRCode func(url, key string)) error {
	// 1. Try existing cookie
	ok, err := c.IsLogin()
	if err != nil {
		return err
	}
	if ok {
		return nil
	}

	// 2. Try refreshing cookie
	if c.RefreshToken != "" {
		if err := c.RefreshCookie(); err == nil {
			ok, err = c.IsLogin()
			if err != nil {
				return err
			}
			if ok {
				return nil
			}
		}
	}

	// 3. Fall back to QR code login
	if onQRCode == nil {
		return errors.New("cookie invalid and no QR code handler provided")
	}
	url, key, err := GenerateQRCode()
	if err != nil {
		return err
	}
	onQRCode(url, key)
	return c.PollLogin(key)
}

// IsLogin checks whether the current cookie is valid by calling the nav API.
func (c *Client) IsLogin() (bool, error) {
	uri := "https://api.bilibili.com/x/web-interface/nav"
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Cookie", c.Cookie)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}
	data := gjson.ParseBytes(body)
	return data.Get("code").Int() == 0, nil
}
