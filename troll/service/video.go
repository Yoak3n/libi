package service

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	util2 "github.com/Yoak3n/gulu/util"
	"github.com/Yoak3n/libi/shared/domain/model/table"
	"troll/internal/util"
	"troll/model"
)

type Video struct {
	Bvid  string
	Avid  int64
	cache string
	name  string
}

func NewVideo(cache, name, bvid string, avid int64) *Video {
	if name == "" {
		fmt.Println("You need to specify a title to save this video's data")
		return nil
	}
	v := &Video{cache: cache, name: name}
	if avid != -1 {
		bvid = util.Avid2Bvid(avid)
		v.Avid = avid
		v.Bvid = bvid
	} else if bvid != "" {
		if strings.HasPrefix(bvid, "http://") || strings.HasPrefix(bvid, "https://") {
			parsed, err := url.Parse(bvid)
			if err != nil {
				fmt.Println("video url parse failed", err)
				return nil
			}
			segments := strings.Split(strings.Trim(parsed.Path, "/"), "/")
			for i, seg := range segments {
				if seg == "video" && i+1 < len(segments) {
					b := segments[i+1]
					if strings.HasPrefix(b, "BV1") && len(b) >= 12 {
						bvid = b
						avid = util.Bvid2Avid(bvid)
					}
				}
			}
		} else if strings.HasPrefix(bvid, "BV1") && len(bvid) >= 12 {
			avid = util.Bvid2Avid(bvid)
		} else {
			bvid = "BV1" + bvid
		}
		v.Avid = avid
		v.Bvid = bvid
	}
	v.Run()
	return v
}

func (v *Video) Run() {
	videoInfo := FetchVideoInfo(v.Bvid, v.name)
	if videoInfo == nil {
		log.Printf("ERROR: Failed to fetch video info for %s", v.Bvid)
		return
	}
	log.Printf("Fetching video: %s", videoInfo.Title)
	start := time.Now()
	videoData := &model.VideoData{
		Avid:        videoInfo.Avid,
		Bvid:        videoInfo.Bvid,
		Title:       videoInfo.Title,
		Cover:       videoInfo.Cover,
		Description: videoInfo.Description,
		Owner:       videoInfo.Owner,
	}
	tracker := NewProgressTracker()
	tracker.RegisterWorker(1)
	videoData.Comments = LazilyGetAllComments(videoInfo.Avid, videoInfo.Review, tracker, 1, videoInfo.Title)

	videoRecord := &table.VideoTable{
		Avid:        videoData.Avid,
		Title:       videoData.Title,
		Bvid:        videoData.Bvid,
		Description: videoData.Description,
		Owner:       videoData.Owner.Uid,
		Topic:       v.name,
	}
	VideoRepo.CreateVideo(videoRecord)
	AddUserByUid(videoData.Owner.Uid)

	if v.cache != "" {
		count := CountComments(videoData)
		out := &model.VideoDataOutput{
			VideoID: v.Bvid,
			Count:   count,
			Data:    videoData,
		}
		jsonData, _ := json.Marshal(out)
		dir := fmt.Sprintf("%s/%s", v.cache, v.name)
		_ = util2.CreateDirNotExists(dir)
		file, err := os.Create(fmt.Sprintf("%s/%s.json", dir, videoData.Title))
		if err == nil {
			file.Write(jsonData)
			file.Close()
		}
	}
	log.Printf("Completed: %s (%v)", videoData.Title, time.Since(start).Round(time.Second))
}
