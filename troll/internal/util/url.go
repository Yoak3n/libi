package util

import (
	"fmt"
	"strings"
)

func AppendParamsToUrl(url string, params map[string]string) string {
	url = url + "?"
	list := make([]string, 0, len(params))
	for k, v := range params {
		list = append(list, fmt.Sprintf("%s=%s", k, v))
	}
	return url + strings.Join(list, "&")
}

var (
	XorCode = int64(23442827791579)
	MaxCode = int64(2251799813685247)
	Charts  = "FcwAPNKTMug3GV5Lj7EJnHpWsx4tb8haYeviqBz6rkCy12mUSDQX9RdoZf"
	PaulNum = int64(58)
)

func swapString(s string, x, y int) string {
	chars := []rune(s)
	chars[x], chars[y] = chars[y], chars[x]
	return string(chars)
}

func Bvid2Avid(bvid string) (avid int64) {
	s := swapString(swapString(bvid, 3, 9), 4, 7)
	bv1 := string([]rune(s)[3:])
	temp := int64(0)
	for _, c := range bv1 {
		idx := strings.IndexRune(Charts, c)
		temp = temp*PaulNum + int64(idx)
	}
	avid = (temp & MaxCode) ^ XorCode
	return
}

func Avid2Bvid(avid int64) (bvid string) {
	arr := [12]string{"B", "V", "1"}
	bvIdx := len(arr) - 1
	temp := (avid | (MaxCode + 1)) ^ XorCode
	for temp > 0 {
		idx := temp % PaulNum
		arr[bvIdx] = string(Charts[idx])
		temp /= PaulNum
		bvIdx--
	}
	raw := strings.Join(arr[:], "")
	bvid = swapString(swapString(raw, 3, 9), 4, 7)
	return
}
