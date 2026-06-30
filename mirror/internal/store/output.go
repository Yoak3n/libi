// Package store 提供分析结果的持久化。
//
// 分析结果以 JSON 文件写入磁盘，不存入数据库。
// 数据库仅用于任务状态追踪（运行中/已完成/失败）。
//
// 目录结构：
//
//	<work_dir>/analyses/
//	  <task_name>_<timestamp>/
//	    task.json               # 任务元信息
//	    videos/
//	      <bvid>.json            # 该视频全部评论的分析结果
//	    controversy/
//	      <bvid>.json            # 该视频的争议因果分析
//	    matrix.json              # 社区关系矩阵（Phase 3）
package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// OutputDir 当前分析的输出目录（由 PrepareOutputDir 设置）
var OutputDir string

// PrepareOutputDir 创建分析结果的输出目录。
// 路径: <workDir>/analyses/<taskName>_<timestamp>/
func PrepareOutputDir(workDir, taskName string) (string, error) {
	timestamp := time.Now().Format("2006-01-02_150405")
	dir := filepath.Join(workDir, "analyses", fmt.Sprintf("%s_%s", taskName, timestamp))

	if err := os.MkdirAll(filepath.Join(dir, "videos"), 0755); err != nil {
		return "", fmt.Errorf("create videos dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "controversy"), 0755); err != nil {
		return "", fmt.Errorf("create controversy dir: %w", err)
	}

	OutputDir = dir
	return dir, nil
}

// ──────────────────────────────────────────────
// Video 分析结果
// ──────────────────────────────────────────────

// CommentOutput 单条评论的完整分析（纯 JSON 友好结构）
type CommentOutput struct {
	Rpid                uint   `json:"rpid"`
	CommentText         string `json:"comment_text"`
	AuthorUID           uint   `json:"author_uid"`
	AuthorName          string `json:"author_name"`
	AffiliatedCommunity string `json:"video_community"`

	// 社区归属
	InferredCommunity   string  `json:"inferred_community"`
	CommunityConfidence float64 `json:"community_confidence"`
	IsCrossCommunity    bool    `json:"is_cross_community"`

	// 专指标 Agent 分数
	VR  *MetricOutput `json:"vr,omitempty"`
	AGI *MetricOutput `json:"agi,omitempty"`
	FI  *MetricOutput `json:"fi,omitempty"`
	CB  *MetricOutput `json:"cb,omitempty"`
	DR  *MetricOutput `json:"dr,omitempty"`
	CI  *MetricOutput `json:"ci,omitempty"`
	EI  *MetricOutput `json:"ei,omitempty"`
}

type MetricOutput struct {
	Score      float64 `json:"score"`
	Confidence float64 `json:"confidence"`
	Evidence   string  `json:"evidence,omitempty"`
}

// VideoOutput 单个视频的完整分析结果
type VideoOutput struct {
	Avid        uint   `json:"avid"`
	Bvid        string `json:"bvid"`
	Title       string `json:"title"`
	CommunityID uint   `json:"community_id"`
	AnalyzedAt  string `json:"analyzed_at"`
	TotalRoot   int    `json:"total_root_comments"`
	Analyzed    int    `json:"analyzed_count"`

	Comments []CommentOutput `json:"comments"`
}

// WriteVideoResult 将视频的分析结果写入 JSON 文件
func WriteVideoResult(outputDir string, vo *VideoOutput) error {
	vo.AnalyzedAt = time.Now().Format(time.RFC3339)
	data, err := json.MarshalIndent(vo, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal video result: %w", err)
	}

	path := filepath.Join(outputDir, "videos", vo.Bvid+".json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write video result: %w", err)
	}
	return nil
}

// ──────────────────────────────────────────────
// Controversy 分析结果
// ──────────────────────────────────────────────

// ControversyOutput 争议分析结果文件格式
type ControversyOutput struct {
	VideoAvid  uint                     `json:"avid"`
	VideoBvid  string                   `json:"bvid"`
	AnalyzedAt string                   `json:"analyzed_at"`
	Topics     []ControversyTopicOutput `json:"topics"`
	Summary    string                   `json:"summary"`
}

type ControversyTopicOutput struct {
	Topic       string   `json:"topic"`
	Comments    []uint   `json:"involved_rpids"`
	Communities []string `json:"communities"`
	Pattern     string   `json:"pattern"`
	TriggerRpid uint     `json:"trigger_rpid,omitempty"`
	Escalation  float64  `json:"escalation_level"`
}

// WriteControversyResult 将争议分析结果写入 JSON 文件
func WriteControversyResult(outputDir string, co *ControversyOutput) error {
	co.AnalyzedAt = time.Now().Format(time.RFC3339)
	data, err := json.MarshalIndent(co, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal controversy result: %w", err)
	}

	path := filepath.Join(outputDir, "controversy", co.VideoBvid+".json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write controversy result: %w", err)
	}
	return nil
}

// ──────────────────────────────────────────────
// Task 元信息
// ──────────────────────────────────────────────

// TaskOutput 任务元信息
type TaskOutput struct {
	Name        string   `json:"name"`
	Communities []string `json:"communities"`
	StartedAt   string   `json:"started_at"`
	TotalVideos int      `json:"total_videos"`
	Status      string   `json:"status"`
}

// WriteTaskMeta 写入任务元信息
func WriteTaskMeta(outputDir string, meta *TaskOutput) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outputDir, "task.json"), data, 0644)
}

// ──────────────────────────────────────────────
