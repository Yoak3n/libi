// Package agent 实现 mirror 的 AI Agent 流水线。
//
// 每个指标一个专指标 Agent（Specialized Metric Agent），
// 每个 Agent 只做一件事、只输出一个量化分数。
package agent

import (
	"context"
	"fmt"
	"time"

	mirrorconfig "mirror/internal/config"
	"mirror/internal/llm"
)

// ──────────────────────────────────────────────
// 基础类型
// ──────────────────────────────────────────────

// AgentType Agent 类型
type AgentType string

const (
	AgentCommunity AgentType = "community" // 社区归属识别
	AgentVR        AgentType = "vr"        // 串门强度
	AgentAGI       AgentType = "agi"       // 攻击性指数
	AgentFI        AgentType = "fi"        // 友善度
	AgentCB        AgentType = "cb"        // 拉踩倾向
	AgentDR        AgentType = "dr"        // 防御反应
	AgentCI        AgentType = "ci"        // 冲突卷入
	AgentEI        AgentType = "ei"        // 情绪烈度
)

// MetricResult 专指标 Agent 的输出
type MetricResult struct {
	Score      float64 `json:"score"`      // 主要分数
	Confidence float64 `json:"confidence"` // 置信度 [0,1]
	Evidence   string  `json:"evidence"`   // 一句话推理摘要
}

// CommunityResult Agent-Community 的完整输出
type CommunityResult struct {
	InferredCommunity string  // 推断的出身社区
	Confidence        float64 // 归属置信度 [0,1]
	Reasoning         string  // 推理摘要
}

// CommentInput 提交给 Agent 的评论上下文
type CommentInput struct {
	Rpid          uint   // 评论 ID
	CommentText   string // 评论原文
	CommentAuthor uint   // 评论者 UID
	AuthorName    string // 评论者昵称
	VideoTitle    string // 视频标题
	VideoAvid     uint   // 视频 avid
	VideoTags     string // 视频标签（逗号分隔）

	// 社区归属（由 Agent-Community 预先算出）
	InferredCommunity   string  // 推断的出身社区
	CommunityConfidence float64 // 归属置信度

	// 视频归属
	AffiliatedCommunity string // 视频归属社区
	IsCrossCommunity    bool   // 是否为跨社区评论

	// 讨论链上下文（可选，CI/DR 需要）
	ParentComment   *CommentInput
	SiblingComments []CommentInput
}

// AgentContext Agent 运行时上下文
type AgentContext struct {
	LLMClient *llm.Client
	Logger    func(format string, args ...interface{})
}

// Agent 统一 Agent 接口
type Agent interface {
	Type() AgentType
	Run(ctx context.Context, input CommentInput, ac *AgentContext) (*MetricResult, error)
}

// ──────────────────────────────────────────────
// Agent 运行结果日志
// ──────────────────────────────────────────────

// AgentRunLog 记录一次 Agent 调用的完整信息（审计用）
type AgentRunLog struct {
	AgentType   AgentType
	Rpid        uint
	Status      string // success | failed | timeout
	DurationMs  int
	Prompt      string
	RawResponse string
	ResultJSON  string
	TokenUsage  *llm.Usage
	Error       string
}

// ──────────────────────────────────────────────
// 辅助函数
// ──────────────────────────────────────────────

// buildLLMClient 从配置构建 LLM 客户端
func buildLLMClient() (*llm.Client, error) {
	cfg, err := loadLLMConfig()
	if err != nil {
		return nil, err
	}
	return llm.NewClient(*cfg), nil
}

// loadLLMConfig 从 mirror 配置加载 LLM 客户端配置
func loadLLMConfig() (*llm.Config, error) {
	if mirrorconfig.Conf == nil {
		return nil, fmt.Errorf("mirror config not initialized")
	}
	apiKey := mirrorconfig.ResolveAPIKey()
	if apiKey == "" {
		return nil, fmt.Errorf("LLM API key not set")
	}
	baseURL := mirrorconfig.ResolveBaseURL()

	return &llm.Config{
		Provider: mirrorconfig.Conf.LLM.Provider,
		APIKey:   apiKey,
		Model:    mirrorconfig.Conf.LLM.Model,
		BaseURL:  baseURL,
		Timeout:  time.Duration(mirrorconfig.Conf.LLM.Timeout) * time.Second,
	}, nil
}
