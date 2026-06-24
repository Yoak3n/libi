package fetch

import (
	"bdanmu/util"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Yoak3n/libi/shared/domain/model/schema"
	"github.com/Yoak3n/libi/shared/package/request"
	"github.com/tidwall/gjson"
)

func GetUserInfo(uid uint) *schema.User {
	count := 0
	for {
		res, err := request.Get("https://api.bilibili.com/x/web-interface/card", fmt.Sprintf("mid=%d", uid))
		if err != nil {
			continue
		}
		result := gjson.ParseBytes(res)
		if code := result.Get("code"); code.Exists() && code.Int() != 0 {
			count += 1
			if count > 5 {
				return nil
			}
			time.Sleep(time.Second)
			continue
		}
		data := result.Get("data")
		u := &schema.User{
			UID:           uint(uid),
			Avatar:        data.Get("card.face").String(),
			Name:          data.Get("card.name").String(),
			Sex:           util.TransSex(data.Get("card.sex").String()),
			FollowerCount: data.Get("follower").Int(),
		}
		return u
	}
}

func GetUserInfoMultiply(uids []uint) []*schema.User {
	users := make([]*schema.User, 0)
	uidsStr := make([]string, 0, len(uids))

	// 简单去重
	seen := make(map[uint]struct{}, len(uids))
	for _, uid := range uids {
		if _, ok := seen[uid]; ok {
			continue
		}
		seen[uid] = struct{}{}
		uidsStr = append(uidsStr, strconv.FormatUint(uint64(uid), 10))
	}

	if len(uidsStr) == 0 {
		return users
	}

	targets := strings.Join(uidsStr, ",")
	count := 0
	for {
		res, err := request.Get("https://api.vc.bilibili.com/account/v1/user/cards", fmt.Sprintf("uids=%s", targets))
		if err != nil {
			continue
		}
		result := gjson.ParseBytes(res)
		if code := result.Get("code"); code.Exists() && code.Int() != 0 {
			count++
			if count > 5 {
				return nil
			}
			time.Sleep(time.Second)
			continue
		}
		for _, v := range result.Get("data").Array() {
			u := &schema.User{
				UID:    uint(v.Get("mid").Int()),
				Avatar: v.Get("face").String(),
				Name:   v.Get("name").String(),
				Sex:    util.TransSex(v.Get("sex").String()),
			}
			users = append(users, u)
		}
		return users
	}
}
