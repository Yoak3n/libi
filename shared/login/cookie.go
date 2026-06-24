package login

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/tidwall/gjson"
)

// CheckCookieValid checks whether the current cookie needs refreshing.
// Returns true if the cookie is still valid.
func (c *Client) CheckCookieValid() bool {
	needRefresh, _, err := c.checkNeedRefresh()
	if err != nil {
		return false
	}
	return !needRefresh
}

// RefreshCookie refreshes the cookie if needed. Returns nil if already valid.
func (c *Client) RefreshCookie() error {
	needRefresh, _, err := c.checkNeedRefresh()
	if err != nil {
		return err
	}
	if !needRefresh {
		return nil
	}

	refreshCsrf, err := c.getRefreshCsrf()
	if err != nil {
		return err
	}

	newRefreshToken, err := c.refreshCookie(refreshCsrf)
	if err != nil {
		return err
	}

	if err := c.commitCookie(); err != nil {
		return err
	}

	c.RefreshToken = newRefreshToken
	c.save()
	return nil
}

func (c *Client) checkNeedRefresh() (bool, int64, error) {
	csrf := getCsrf(c.Cookie)
	uri := "https://passport.bilibili.com/x/passport-login/web/cookie/info?csrf=" + csrf

	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return false, 0, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Cookie", c.Cookie)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, 0, err
	}
	data := gjson.ParseBytes(body)
	if data.Get("code").Int() != 0 {
		return false, 0, errors.New("check cookie failed: " + data.Get("message").String())
	}
	if data.Get("data.refresh").Bool() {
		return true, data.Get("data.timestamp").Int(), nil
	}
	return false, 0, nil
}

func (c *Client) getRefreshCsrf() (string, error) {
	correspondPath, err := getCorrespondPath(time.Now().UnixMilli())
	if err != nil {
		return "", err
	}
	uri := "https://www.bilibili.com/correspond/1/" + correspondPath

	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Cookie", c.Cookie)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	dom, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", err
	}
	return dom.Find("#1-name").Text(), nil
}

func (c *Client) refreshCookie(refreshCsrf string) (string, error) {
	csrf := getCsrf(c.Cookie)
	postData := url.Values{}
	postData.Set("refresh_token", c.RefreshToken)
	postData.Set("source", "main_page")
	postData.Set("refresh_csrf", refreshCsrf)
	postData.Set("csrf", csrf)

	req, err := http.NewRequest("POST",
		"https://passport.bilibili.com/x/passport-login/web/cookie/refresh",
		strings.NewReader(postData.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Cookie", c.Cookie)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	data := gjson.ParseBytes(body)
	if data.Get("code").Int() != 0 {
		return "", errors.New(data.Get("message").String())
	}

	c.Cookie = parseCookies(resp.Header)
	return data.Get("data.refresh_token").String(), nil
}

func (c *Client) commitCookie() error {
	csrf := getCsrf(c.Cookie)
	postData := url.Values{}
	postData.Set("csrf", csrf)
	postData.Set("refresh_token", c.RefreshToken)

	req, err := http.NewRequest("POST",
		"https://passport.bilibili.com/x/passport-login/web/confirm/refresh",
		strings.NewReader(postData.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Cookie", c.Cookie)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	data := gjson.ParseBytes(body)
	if data.Get("code").Int() != 0 {
		return errors.New(data.Get("message").String())
	}
	return nil
}
