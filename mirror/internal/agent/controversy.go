package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"mirror/internal/llm"
	"mirror/internal/store"
)

// ──────────────────────────────────────────────
// 争议分析结果类型
// ──────────────────────────────────────────────

type ControversyTopic struct {
	Topic       string
	Comments    []uint
	SourceComms []string
	Pattern     string // single_invasion | bilateral | internal_spillover
	TriggerRpid uint
	Escalation  float64
}

type ControversyResult struct {
	VideoAvid uint               `json:"video_avid"`
	Topics    []ControversyTopic `json:"topics"`
	Summary   string             `json:"summary"`
}

// ──────────────────────────────────────────────
// 争议因果分析器
// ──────────────────────────────────────────────

type ControversyAnalyzer struct {
	llmClient *llm.Client
	outputDir string
}

func NewControversyAnalyzer(client *llm.Client, outputDir string) *ControversyAnalyzer {
	return &ControversyAnalyzer{llmClient: client, outputDir: outputDir}
}

// Analyze 分析指定视频的争议情况，从已写入的 JSON 文件读取数据
func (ca *ControversyAnalyzer) Analyze(ctx context.Context, avid uint) (*ControversyResult, error) {
	log.Printf("[Controversy] Analyzing avid=%d", avid)

	// 读取 JSON 文件获取已分析的评论（从 outputDir/videos/<bvid>.json）
	// 实际调用方需传入已知的 bvid 或通过 loadCrossComments 读取
	// 这里简化：由 RunControversyAnalysis 传入所有数据
	return nil, fmt.Errorf("use RunControversyAnalysis with explicit data")
}

// AnalyzeFromOutput 从 VideoOutput 分析争议
func (ca *ControversyAnalyzer) AnalyzeFromOutput(ctx context.Context, vo *store.VideoOutput) (*ControversyResult, error) {
	if len(vo.Comments) == 0 {
		return nil, nil
	}

	// 只取跨社区评论
	var crossComments []store.CommentOutput
	for _, c := range vo.Comments {
		if c.IsCrossCommunity {
			crossComments = append(crossComments, c)
		}
	}
	if len(crossComments) == 0 {
		return nil, nil
	}

	// 调用 LLM
	input := controversyLLMInput{}
	for _, c := range crossComments {
		input.Comments = append(input.Comments, controversyCommentInput{
			Rpid:            c.Rpid,
			Text:            truncate(c.CommentText, 200),
			AuthorCommunity: c.InferredCommunity,
			VideoCommunity:  c.AffiliatedCommunity,
			AGIScore:        scoreOrZero(c.AGI),
			FIScore:         scoreOrZero(c.FI),
		})
	}

	var output controversyLLMOutput
	req := llm.ChatRequest{
		Messages: []llm.Message{
			{Role: "system", Content: controversySystemPrompt},
			{Role: "user", Content: fmt.Sprintf("分析以下跨社区评论的争议模式：%s", mustJSON(input))},
		},
		Temperature: 0.2,
		MaxTokens:   2048,
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	if _, err := ca.llmClient.ChatJSON(req, &output); err != nil {
		return nil, err
	}

	result := &ControversyResult{VideoAvid: vo.Avid, Summary: output.OverallSummary}
	for _, t := range output.Topics {
		result.Topics = append(result.Topics, ControversyTopic{
			Topic:       t.Topic,
			Comments:    t.InvolvedRpids,
			SourceComms: t.Communities,
			Pattern:     t.Pattern,
			TriggerRpid: t.TriggerRpid,
			Escalation:  t.EscalationLevel,
		})
	}

	// 写文件
	ca.saveResult(vo.Bvid, result)
	return result, nil
}

func scoreOrZero(m *store.MetricOutput) float64 {
	if m == nil {
		return 0
	}
	return m.Score
}

func (ca *ControversyAnalyzer) saveResult(bvid string, result *ControversyResult) {
	if ca.outputDir == "" {
		return
	}
	co := &store.ControversyOutput{
		VideoAvid: result.VideoAvid,
		VideoBvid: bvid,
		Topics:    make([]store.ControversyTopicOutput, len(result.Topics)),
		Summary:   result.Summary,
	}
	for i, t := range result.Topics {
		co.Topics[i] = store.ControversyTopicOutput{
			Topic:       t.Topic,
			Comments:    t.Comments,
			Communities: t.SourceComms,
			Pattern:     t.Pattern,
			TriggerRpid: t.TriggerRpid,
			Escalation:  t.Escalation,
		}
	}
	if err := store.WriteControversyResult(ca.outputDir, co); err != nil {
		log.Printf("[Controversy] write result failed: %v", err)
	}
}

// ──────────────────────────────────────────────
// LLM 输入/输出
// ──────────────────────────────────────────────

type controversyLLMInput struct {
	Comments []controversyCommentInput `json:"comments"`
}

type controversyCommentInput struct {
	Rpid            uint    `json:"rpid"`
	Text            string  `json:"text"`
	AuthorCommunity string  `json:"author_community"`
	VideoCommunity  string  `json:"video_community"`
	AGIScore        float64 `json:"agi_score"`
	FIScore         float64 `json:"fi_score"`
}

type controversyLLMOutput struct {
	Topics []struct {
		Topic           string   `json:"topic"`
		InvolvedRpids   []uint   `json:"involved_rpids"`
		Communities     []string `json:"communities"`
		Pattern         string   `json:"pattern"`
		TriggerRpid     uint     `json:"trigger_rpid,omitempty"`
		EscalationLevel float64  `json:"escalation_level"`
		Summary         string   `json:"summary"`
	} `json:"topics"`
	OverallSummary string `json:"overall_summary"`
}

const controversySystemPrompt = `你是一个分析Bilibili评论区「跨社区争议」的专家。
你的任务是从一组跨社区评论中，识别出争议议题、对立模式和因果链。

输入：每条评论包含：
- rpid: 评论ID
- text: 评论原文
- author_community: 评论者出身的游戏社区
- video_community: 视频归属的游戏社区
- agi_score: 攻击性指数 [0,1]
- fi_score: 友善度 [0,1]

你需要识别：
1. 争议议题（大家在争论什么）
2. 对立模式：
   - single_invasion: 全部是外部社区的人在批评
   - bilateral: 双方社区成员互相攻击
   - internal_spillover: 视频所属社区的内部矛盾暴露在外部
3. 触发源（哪条评论最先引发冲突）
4. 冲突升级程度 [0,1]

输出JSON：
{
  "topics": [
    {
      "topic": "争议议题描述",
      "involved_rpids": [参与评论ID],
      "communities": ["涉及社区"],
      "pattern": "single_invasion|bilateral|internal_spillover",
      "trigger_rpid": 触发源rpid,
      "escalation_level": 0.0-1.0,
      "summary": "简要描述"
    }
  ],
  "overall_summary": "整体概括"
}`

// ──────────────────────────────────────────────
// 便捷入口
// ──────────────────────────────────────────────

// RunControversyAnalysis 读取视频分析结果 JSON，执行争议分析
func RunControversyAnalysis(ctx context.Context, vo *store.VideoOutput, outputDir string) error {
	client, err := buildLLMClient()
	if err != nil {
		return fmt.Errorf("build LLM client: %w", err)
	}
	analyzer := NewControversyAnalyzer(client, outputDir)
	_, err = analyzer.AnalyzeFromOutput(ctx, vo)
	return err
}

func mustJSON(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
