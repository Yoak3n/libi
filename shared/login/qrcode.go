package login

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

// GenerateQRCode fetches a QR code login URL from Bilibili.
// Returns the login URL (to display as QR code) and the login key (to poll with).
func GenerateQRCode() (url, key string, err error) {
	uri := "https://passport.bilibili.com/x/passport-login/web/qrcode/generate"
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}
	data := gjson.ParseBytes(body)
	url = data.Get("data.url").String()
	key = data.Get("data.qrcode_key").String()
	if url == "" || key == "" {
		return "", "", errors.New("failed to generate QR code")
	}
	return url, key, nil
}

// PollLogin polls the Bilibili QR code login endpoint until the user scans the code.
// On success, updates c.Cookie and c.RefreshToken and persists via callback.
func (c *Client) PollLogin(loginKey string) error {
	uri := "https://passport.bilibili.com/x/passport-login/web/qrcode/poll?qrcode_key=" + loginKey

	for {
		req, err := http.NewRequest("GET", uri, nil)
		if err != nil {
			return err
		}
		req.Header.Set("User-Agent", userAgent)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return err
		}

		data := gjson.ParseBytes(body)
		if data.Get("data.url").String() != "" {
			cookie := parseCookies(resp.Header)
			c.Cookie = cookie
			c.RefreshToken = data.Get("data.refresh_token").String()
			c.save()
			return nil
		}

		time.Sleep(3 * time.Second)
	}
}

// parseCookies extracts Bilibili auth cookies from Set-Cookie headers.
func parseCookies(header http.Header) string {
	cookies := make(map[string]string)
	for _, v := range header["Set-Cookie"] {
		kv, _, _ := strings.Cut(v, ";")
		if key, val, ok := strings.Cut(kv, "="); ok {
			cookies[key] = val
		}
	}
	return "DedeUserID=" + cookies["DedeUserID"] +
		";DedeUserID__ckMd5=" + cookies["DedeUserID__ckMd5"] +
		";Expires=" + cookies["Expires"] +
		";SESSDATA=" + cookies["SESSDATA"] +
		";bili_jct=" + cookies["bili_jct"] + ";"
}
