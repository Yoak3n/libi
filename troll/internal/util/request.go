package util

import (
	"io"
	"math"
	"net/http"
	"net/url"
	"time"

	"github.com/Yoak3n/libi/shared/config"
)

func ClientWithProxy() *http.Client {
	client := &http.Client{Timeout: 30 * time.Second}
	if config.Conf != nil && config.Conf.Proxy != "" {
		parsed, err := url.Parse(config.Conf.Proxy)
		if err == nil {
			client.Transport = &http.Transport{Proxy: http.ProxyURL(parsed)}
		}
	}
	return client
}

func GetRequestWithCookie(addr string, cookie string) *http.Request {
	uri, err := url.Parse(addr)
	if err != nil {
		return nil
	}
	if err := Sign(uri); err != nil {
		return nil
	}
	req := &http.Request{
		Method: "GET",
		URL:    uri,
		Header: http.Header{
			"Cookie":     []string{cookie},
			"User-Agent": []string{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"},
		},
	}
	return req
}

func RequestGetWithAll(addr string, cookie string) []byte {
	client := ClientWithProxy()
	req := GetRequestWithCookie(addr, cookie)
	if req == nil {
		return nil
	}
	res, err := client.Do(req)
	if err != nil {
		return requestRetry(req, 1)
	}
	if res.StatusCode != 200 {
		res.Body.Close()
		return requestRetry(req, 1)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	return body
}

func requestRetry(req *http.Request, count int) []byte {
	if count >= 10 {
		return nil
	}
	time.Sleep(time.Duration(math.Min(300, math.Pow(2, float64(count)))) * time.Second)
	client := ClientWithProxy()
	res, err := client.Do(req)
	if err != nil {
		return requestRetry(req, count+1)
	}
	if res.StatusCode != 200 {
		res.Body.Close()
		return requestRetry(req, count+1)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	return body
}
