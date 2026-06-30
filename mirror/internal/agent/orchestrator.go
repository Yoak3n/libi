package agent

import (
	"context"
	"log"
	"time"

	"mirror/internal/store"
)

// VideoTask 单个视频的分析任务参数
type VideoTask struct {
	Avid     uint
	Bvid     string
	Title    string
	Comments []CommentInput
}

// Orchestrator 管理整个分析流程
type Orchestrator struct {
	taskName    string
	communities []string
	videoTasks  []VideoTask
	outputDir   string
	startedAt   time.Time
}

func NewOrchestrator(taskName string, communities []string) *Orchestrator {
	return &Orchestrator{
		taskName:    taskName,
		communities: communities,
	}
}

func (o *Orchestrator) SetOutputDir(dir string) {
	o.outputDir = dir
}

func (o *Orchestrator) AddVideo(task VideoTask) {
	o.videoTasks = append(o.videoTasks, task)
}

func (o *Orchestrator) Run(ctx context.Context) error {
	o.startedAt = time.Now()
	log.Printf("[Orchestrator] Starting: %s (%d videos)", o.taskName, len(o.videoTasks))

	// 写 task.json
	if o.outputDir != "" {
		store.WriteTaskMeta(o.outputDir, &store.TaskOutput{
			Name:        o.taskName,
			Communities: o.communities,
			StartedAt:   o.startedAt.Format(time.RFC3339),
			TotalVideos: len(o.videoTasks),
			Status:      "running",
		})
	}

	success := 0
	for i, vt := range o.videoTasks {
		log.Printf("[Orchestrator] Video %d/%d: %s", i+1, len(o.videoTasks), vt.Title)
		va := NewVideoAgent(vt.Avid, vt.Bvid, vt.Title)
		va.OutputDir = o.outputDir
		if err := va.Run(ctx, vt.Comments); err != nil {
			log.Printf("[Orchestrator] %s failed: %v", vt.Bvid, err)
			continue
		}
		success++
	}

	if o.outputDir != "" {
		store.WriteTaskMeta(o.outputDir, &store.TaskOutput{
			Name:        o.taskName,
			Communities: o.communities,
			StartedAt:   o.startedAt.Format(time.RFC3339),
			TotalVideos: len(o.videoTasks),
			Status:      "completed",
		})
	}

	elapsed := time.Since(o.startedAt).Round(time.Second)
	log.Printf("[Orchestrator] Done: %d/%d videos in %s → %s",
		success, len(o.videoTasks), elapsed, o.outputDir)
	return nil
}
