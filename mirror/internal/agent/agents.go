package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"mirror/internal/llm"
)

// ──────────────────────────────────────────────
// 通用 Agent 执行器
// ──────────────────────────────────────────────

// agentRunner 封装一次 LLM 调用的通用逻辑
type agentRunner struct {
	agentType    AgentType
	systemPrompt string
	buildUserMsg func(CommentInput) string
	timeout      time.Duration
}

func (r *agentRunner) run(ctx context.Context, input CommentInput, ac *AgentContext, resultTarget interface{}) (*llm.Usage, error) {
	req := llm.ChatRequest{
		Messages: []llm.Message{
			{Role: "system", Content: r.systemPrompt},
			{Role: "user", Content: r.buildUserMsg(input)},
		},
		Temperature: 0.1, // 低温度确保一致性
		MaxTokens:   512,
	}

	// 设置超时
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	usage, err := ac.LLMClient.ChatJSON(req, resultTarget)
	if err != nil {
		return nil, fmt.Errorf("[%s] LLM call failed: %w", r.agentType, err)
	}
	return usage, nil
}

// ──────────────────────────────────────────────
// Agent-Community: 社区归属识别
// ──────────────────────────────────────────────

// communityResult 解析 Agent-Community 的 LLM JSON 输出
type communityResultRaw struct {
	InferredCommunity string  `json:"inferred_community"`
	Confidence        float64 `json:"confidence"`
	Reasoning         string  `json:"reasoning"`
}

// CommunityAgent 社区归属识别 Agent
type CommunityAgent struct {
	runner agentRunner
}

func NewCommunityAgent() *CommunityAgent {
	return &CommunityAgent{
		runner: agentRunner{
			agentType:    AgentCommunity,
			systemPrompt: CommunitySystemPrompt,
			buildUserMsg: func(input CommentInput) string {
				return CommunityUserTemplate(input.CommentText, input.VideoTitle, input.VideoTags, input.AffiliatedCommunity)
			},
			timeout: 30 * time.Second,
		},
	}
}

func (a *CommunityAgent) Type() AgentType { return AgentCommunity }

func (a *CommunityAgent) Run(ctx context.Context, input CommentInput, ac *AgentContext) (*CommunityResult, error) {
	var raw communityResultRaw
	_, err := a.runner.run(ctx, input, ac, &raw)
	if err != nil {
		return nil, err
	}

	return &CommunityResult{
		InferredCommunity: raw.InferredCommunity,
		Confidence:        raw.Confidence,
		Reasoning:         raw.Reasoning,
	}, nil
}

// ──────────────────────────────────────────────
// 通用专指标 Agent 工厂
// ──────────────────────────────────────────────

// metricAgent 通用的专指标 Agent 实现
type metricAgent struct {
	agentType    AgentType
	systemPrompt string
	buildUserMsg func(CommentInput) string
	timeout      time.Duration
	// extractScore 从解析后的 JSON 中提取分数
	extractScore func(raw map[string]interface{}) *MetricResult
}

func (a *metricAgent) Type() AgentType { return a.agentType }

func (a *metricAgent) Run(ctx context.Context, input CommentInput, ac *AgentContext) (*MetricResult, error) {
	// 非跨社区评论只跑基础分数
	if input.IsCrossCommunity {
		// 全量跑
	} else if a.agentType != AgentEI {
		// 非跨社区评论只跑 EI（情绪烈度对所有评论都有意义）
		// 其他指标跳过
		return &MetricResult{Score: 0, Confidence: 0, Evidence: "本社区评论，跳过跨社区指标"}, nil
	}

	req := llm.ChatRequest{
		Messages: []llm.Message{
			{Role: "system", Content: a.systemPrompt},
			{Role: "user", Content: a.buildUserMsg(input)},
		},
		Temperature: 0.1,
		MaxTokens:   512,
	}

	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	var raw map[string]interface{}
	_, err := ac.LLMClient.ChatJSON(req, &raw)
	if err != nil {
		return nil, fmt.Errorf("[%s] LLM call failed: %w", a.agentType, err)
	}

	result := a.extractScore(raw)
	if result == nil {
		return nil, fmt.Errorf("[%s] failed to extract score from: %v", a.agentType, raw)
	}

	return result, nil
}

// ──────────────────────────────────────────────
// 分数提取辅助
// ──────────────────────────────────────────────

func getFloat(raw map[string]interface{}, key string) float64 {
	v, ok := raw[key]
	if !ok {
		return 0
	}
	f, _ := v.(float64)
	return f
}

func getString(raw map[string]interface{}, key string) string {
	v, ok := raw[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

func getStringSlice(raw map[string]interface{}, key string) []string {
	v, ok := raw[key]
	if !ok {
		return nil
	}
	// JSON unmarshal into interface{} gives []interface{} for arrays
	if iface, ok := v.([]interface{}); ok {
		result := make([]string, len(iface))
		for i, item := range iface {
			result[i], _ = item.(string)
		}
		return result
	}
	return nil
}

func toJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// ──────────────────────────────────────────────
// Agent 工厂：根据类型创建对应的专指标 Agent
// ──────────────────────────────────────────────

// NewMetricAgent 创建一个专指标 Agent
func NewMetricAgent(agentType AgentType) Agent {
	switch agentType {
	case AgentVR:
		return &metricAgent{
			agentType:    AgentVR,
			systemPrompt: VRSystemPrompt,
			buildUserMsg: func(input CommentInput) string {
				return VRUserTemplate(input.CommentText, input.InferredCommunity, input.AffiliatedCommunity)
			},
			timeout: 20 * time.Second,
			extractScore: func(raw map[string]interface{}) *MetricResult {
				return &MetricResult{
					Score:      getFloat(raw, "vr_score"),
					Confidence: getFloat(raw, "confidence"),
					Evidence:   getString(raw, "evidence"),
				}
			},
		}
	case AgentAGI:
		return &metricAgent{
			agentType:    AgentAGI,
			systemPrompt: AGISystemPrompt,
			buildUserMsg: func(input CommentInput) string {
				return VRUserTemplate(input.CommentText, input.InferredCommunity, input.AffiliatedCommunity)
			},
			timeout: 20 * time.Second,
			extractScore: func(raw map[string]interface{}) *MetricResult {
				return &MetricResult{
					Score:      getFloat(raw, "agi_score"),
					Confidence: getFloat(raw, "confidence"),
					Evidence: fmt.Sprintf("hostility=%.2f sarcasm=%.2f trolling=%.2f | %s",
						getFloat(raw, "hostility"), getFloat(raw, "sarcasm"), getFloat(raw, "trolling"), getString(raw, "evidence")),
				}
			},
		}
	case AgentFI:
		return &metricAgent{
			agentType:    AgentFI,
			systemPrompt: FISystemPrompt,
			buildUserMsg: func(input CommentInput) string {
				return VRUserTemplate(input.CommentText, input.InferredCommunity, input.AffiliatedCommunity)
			},
			timeout: 20 * time.Second,
			extractScore: func(raw map[string]interface{}) *MetricResult {
				return &MetricResult{
					Score:      getFloat(raw, "fi_score"),
					Confidence: getFloat(raw, "confidence"),
					Evidence: fmt.Sprintf("praise=%.2f constructive=%.2f welcome=%.2f | %s",
						getFloat(raw, "praise"), getFloat(raw, "constructive"), getFloat(raw, "welcome"), getString(raw, "evidence")),
				}
			},
		}
	case AgentCB:
		return &metricAgent{
			agentType:    AgentCB,
			systemPrompt: CBSystemPrompt,
			buildUserMsg: func(input CommentInput) string {
				return VRUserTemplate(input.CommentText, input.InferredCommunity, input.AffiliatedCommunity)
			},
			timeout: 20 * time.Second,
			extractScore: func(raw map[string]interface{}) *MetricResult {
				targets := getStringSlice(raw, "comparison_targets")
				evidence := getString(raw, "evidence")
				if len(targets) > 0 {
					evidence = fmt.Sprintf("targets=%v | %s", targets, evidence)
				}
				return &MetricResult{
					Score:      getFloat(raw, "cb_score"),
					Confidence: getFloat(raw, "confidence"),
					Evidence:   evidence,
				}
			},
		}
	case AgentDR:
		return &metricAgent{
			agentType:    AgentDR,
			systemPrompt: DRSystemPrompt,
			buildUserMsg: func(input CommentInput) string {
				return VRUserTemplate(input.CommentText, input.InferredCommunity, input.AffiliatedCommunity)
			},
			timeout: 20 * time.Second,
			extractScore: func(raw map[string]interface{}) *MetricResult {
				evidence := getString(raw, "evidence")
				if trigger := getString(raw, "triggered_by"); trigger != "" {
					evidence = fmt.Sprintf("triggered_by: %s | %s", trigger, evidence)
				}
				return &MetricResult{
					Score:      getFloat(raw, "dr_score"),
					Confidence: getFloat(raw, "confidence"),
					Evidence:   evidence,
				}
			},
		}
	case AgentCI:
		return &metricAgent{
			agentType:    AgentCI,
			systemPrompt: CISystemPrompt,
			buildUserMsg: func(input CommentInput) string {
				return VRUserTemplate(input.CommentText, input.InferredCommunity, input.AffiliatedCommunity)
			},
			timeout: 20 * time.Second,
			extractScore: func(raw map[string]interface{}) *MetricResult {
				return &MetricResult{
					Score:      getFloat(raw, "ci_score"),
					Confidence: getFloat(raw, "confidence"),
					Evidence: fmt.Sprintf("turns=%.0f escalation=%.2f | %s",
						getFloat(raw, "turn_count"), getFloat(raw, "escalation"), getString(raw, "evidence")),
				}
			},
		}
	case AgentEI:
		return &metricAgent{
			agentType:    AgentEI,
			systemPrompt: EISystemPrompt,
			buildUserMsg: func(input CommentInput) string {
				return VRUserTemplate(input.CommentText, input.InferredCommunity, input.AffiliatedCommunity)
			},
			timeout: 20 * time.Second,
			extractScore: func(raw map[string]interface{}) *MetricResult {
				return &MetricResult{
					Score:      getFloat(raw, "ei_score"),
					Confidence: getFloat(raw, "confidence"),
					Evidence:   fmt.Sprintf("valence=%.2f | %s", getFloat(raw, "valence"), getString(raw, "evidence")),
				}
			},
		}
	default:
		panic(fmt.Sprintf("unknown agent type: %s", agentType))
	}
}
