package util

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func Sign(u *url.URL) error {
	return wbiKeys.Sign(u)
}

var wbiKeys WbiKeys

type WbiKeys struct {
	Img            string
	Sub            string
	Mixin          string
	lastUpdateTime time.Time
}

func (wk *WbiKeys) Sign(u *url.URL) error {
	if err := wk.update(false); err != nil {
		return err
	}
	values := u.Query()
	values = removeUnwantedChars(values, '!', '\'', '(', ')', '*')
	values.Set("wts", strconv.FormatInt(time.Now().Unix(), 10))
	hash := md5.Sum([]byte(values.Encode() + wk.Mixin))
	values.Set("w_rid", hex.EncodeToString(hash[:]))
	u.RawQuery = values.Encode()
	return nil
}

func (wk *WbiKeys) update(purge bool) error {
	if !purge && time.Since(wk.lastUpdateTime) < time.Hour {
		return nil
	}
	resp, err := http.Get("https://api.bilibili.com/x/web-interface/nav")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var nav struct {
		Code int `json:"code"`
		Data struct {
			WbiImg struct {
				ImgUrl string `json:"img_url"`
				SubUrl string `json:"sub_url"`
			} `json:"wbi_img"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &nav); err != nil {
		return err
	}
	if nav.Code != 0 && nav.Code != -101 {
		return fmt.Errorf("unexpected code: %d", nav.Code)
	}
	imgParts := strings.Split(nav.Data.WbiImg.ImgUrl, "/")
	subParts := strings.Split(nav.Data.WbiImg.SubUrl, "/")
	wk.Img = strings.TrimSuffix(imgParts[len(imgParts)-1], ".png")
	wk.Sub = strings.TrimSuffix(subParts[len(subParts)-1], ".png")
	wk.mixin()
	wk.lastUpdateTime = time.Now()
	return nil
}

func (wk *WbiKeys) mixin() {
	var mixin [32]byte
	wbi := wk.Img + wk.Sub
	for i := range mixin {
		mixin[i] = wbi[mixinKeyEncTab[i]]
	}
	wk.Mixin = string(mixin[:])
}

var mixinKeyEncTab = [...]int{
	46, 47, 18, 2, 53, 8, 23, 32,
	15, 50, 10, 31, 58, 3, 45, 35,
	27, 43, 5, 49, 33, 9, 42, 19,
	29, 28, 14, 39, 12, 38, 41, 13,
	37, 48, 7, 16, 24, 55, 40, 61,
	26, 17, 0, 1, 60, 51, 30, 4,
	22, 25, 54, 21, 56, 59, 6, 63,
	57, 62, 11, 36, 20, 34, 44, 52,
}

func removeUnwantedChars(v url.Values, chars ...byte) url.Values {
	b := []byte(v.Encode())
	for _, c := range chars {
		b = bytes.ReplaceAll(b, []byte{c}, nil)
	}
	s, _ := url.ParseQuery(string(b))
	return s
}
