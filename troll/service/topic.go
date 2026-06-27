package service

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	util2 "github.com/Yoak3n/gulu/util"
	"github.com/Yoak3n/libi/shared/domain/model/table"
	"troll/model"
)

type Topic struct {
	Name    string
	KeyWord []string
	Videos  []model.VideoData
	cache   string
	wg      sync.WaitGroup
	jobs    chan model.VideoData
}

func NewTopic(cache, name string, keywords []string) *Topic {
	now := time.Now()
	t := &Topic{
		Name:    name,
		KeyWord: keywords,
		cache:   cache,
		jobs:    make(chan model.VideoData, 10),
	}
	t.fetchVideos()
	log.Printf("%s cost %vmin", t.Name, time.Since(now).Minutes())
	return t
}

func (t *Topic) fetchVideos() {
	var sb strings.Builder
	sb.WriteString(t.KeyWord[0])
	for i := 1; i < len(t.KeyWord); i++ {
		sb.WriteByte(',')
		sb.WriteString(t.KeyWord[i])
	}
	videos := SearchVideoOfTopic(sb.String(), 1)
	if videos == nil {
		return
	}

	for i := 1; i <= 2; i++ {
		t.wg.Add(1)
		go t.worker(i)
	}
	go func() {
		for _, v := range videos {
			t.jobs <- v
		}
		close(t.jobs)
	}()
	t.wg.Wait()
}

func (t *Topic) worker(id int) {
	defer t.wg.Done()
	for v := range t.jobs {
		log.Printf("Worker %d fetching video: %s", id, v.Title)
		start := time.Now()
		videoData := &model.VideoData{
			Avid:        v.Avid,
			Bvid:        v.Bvid,
			Title:       v.Title,
			Cover:       v.Cover,
			Description: v.Description,
			Owner:       v.Owner,
		}
		videoData.Comments = LazilyGetAllComments(v.Avid, v.Review)

		videoRecord := &table.VideoTable{
			Avid:        v.Avid,
			Title:       v.Title,
			Bvid:        v.Bvid,
			Cover:       v.Cover,
			Description: v.Description,
			Owner:       v.Owner.Uid,
			Topic:       t.Name,
		}
		VideoRepo.CreateVideo(videoRecord)
		AddUserByUid(v.Owner.Uid)

		if t.cache != "" {
			t.saveCache(videoData)
		}
		log.Printf("Worker %d completed: %s (%v)", id, v.Title, time.Since(start).Round(time.Second))
	}
}

func (t *Topic) saveCache(v *model.VideoData) {
	count := CountComments(v)
	out := &model.VideoDataOutput{
		VideoID: v.Bvid,
		Count:   count,
		Data:    v,
	}
 jsonData, _ := json.Marshal(out)
	dir := fmt.Sprintf("%s/%s", t.cache, t.Name)
	_ = util2.CreateDirNotExists(dir)
	file, err := os.Create(fmt.Sprintf("%s/%s.json", dir, v.Title))
	if err != nil {
		return
	}
	defer file.Close()
	file.Write(jsonData)
}
