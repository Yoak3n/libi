package util

import (
	"regexp"
	"strings"
)

var charReplacements = map[string]string{
	":":  "：",
	"<":  "＜",
	">":  "＞",
	"\"": "＂",
	"|":  "｜",
	"?":  "？",
	"*":  "＊",
	"\\": "＼",
	"/":  "／",
}

func SanitizeFilename(fn string) string {
	for k, v := range charReplacements {
		fn = strings.ReplaceAll(fn, k, v)
	}
	return fn
}

func ExtractContentWithinTag(content string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	return re.ReplaceAllString(content, "")
}
