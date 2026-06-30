package agent

import (
	"context"
	"fmt"
	"log"
	"sync"

	mirrorconfig "mirror/internal/config"
	"mirror/internal/llm"
	"mirror/internal/store"
)

// VideoAgent 负责一个视频下所有评论的完整分析流程
type VideoAgent struct {
	Avid       uint
	Bvid       string
	VideoTitle string
	OutputDir  string

	llmClient    *llm.Client
	communityAg  *CommunityAgent
	metricAgents map[AgentType]Agent
	mu           sync.Mutex
}

func NewVideoAgent(avid uint, bvid, title string) *VideoAgent {
	return &VideoAgent{
		Avid:         avid,
		Bvid:         bvid,
		VideoTitle:   title,
		metricAgents: make(map[AgentType]Agent),
	}
}

func (va *VideoAgent) Run(ctx context.Context, comments []CommentInput) error {
	client, err := buildLLMClient()
	if err != nil {
		return fmt.Errorf("build LLM client: %w", err)
	}
	va.llmClient = client
	va.communityAg = NewCommunityAgent()

	ac := &AgentContext{LLMClient: client, Logger: log.Printf}

	// 并发分析评论
	var wg sync.WaitGroup
	sem := make(chan struct{}, mirrorconfig.Conf.Agent.ParallelCount)
	var mu sync.Mutex
	outputs := make([]store.CommentOutput, 0, len(comments))

	for i := range comments {
		wg.Add(1)
		sem <- struct{}{}
		go func(input CommentInput) {
			defer wg.Done()
			defer func() { <-sem }()
			out := va.analyzeComment(ctx, input, ac)
			if out != nil {
				mu.Lock()
				outputs = append(outputs, *out)
				mu.Unlock()
			}
		}(comments[i])
	}
	wg.Wait()

	// 写 JSON 文件
	vo := &store.VideoOutput{
		Avid:      va.Avid,
		Bvid:      va.Bvid,
		Title:     va.VideoTitle,
		TotalRoot: len(comments),
		Analyzed:  len(outputs),
		Comments:  outputs,
	}
	if err := store.WriteVideoResult(va.OutputDir, vo); err != nil {
		log.Printf("[VA] write result failed: %v", err)
	}

	log.Printf("[VA] %d/%d comments → %s/videos/%s.json",
		len(outputs), len(comments), va.OutputDir, va.Bvid)
	return nil
}

func (va *VideoAgent) analyzeComment(ctx context.Context, input CommentInput, ac *AgentContext) *store.CommentOutput {
	// Step 1: 社区归属识别
	cr, err := va.communityAg.Run(ctx, input, ac)
	if err != nil {
		log.Printf("[VA] Agent-Community failed rpid=%d: %v", input.Rpid, err)
		return nil
	}

	input.InferredCommunity = cr.InferredCommunity
	input.CommunityConfidence = cr.Confidence
	input.IsCrossCommunity = cr.InferredCommunity != input.AffiliatedCommunity

	if cr.Confidence < mirrorconfig.Conf.CommunityConfidenceThreshold {
		input.IsCrossCommunity = false
	}

	out := &store.CommentOutput{
		Rpid:                input.Rpid,
		CommentText:         input.CommentText,
		AuthorUID:           input.CommentAuthor,
		AuthorName:          input.AuthorName,
		AffiliatedCommunity: input.AffiliatedCommunity,
		InferredCommunity:   cr.InferredCommunity,
		CommunityConfidence: cr.Confidence,
		IsCrossCommunity:    input.IsCrossCommunity,
	}

	// Step 2: 专指标 Agent
	metricTypes := []AgentType{AgentVR, AgentAGI, AgentFI, AgentCB, AgentDR, AgentCI, AgentEI}
	for _, at := range metricTypes {
		if !input.IsCrossCommunity && at != AgentEI {
			continue
		}
		a := va.getMetricAgent(at)
		r, err := a.Run(ctx, input, ac)
		if err != nil {
			log.Printf("[VA] Agent-%s failed rpid=%d: %v", at, input.Rpid, err)
			continue
		}
		assignMetric(out, at, r)
	}
	return out
}

func assignMetric(out *store.CommentOutput, at AgentType, r *MetricResult) {
	m := &store.MetricOutput{Score: r.Score, Confidence: r.Confidence, Evidence: r.Evidence}
	switch at {
	case AgentVR:
		out.VR = m
	case AgentAGI:
		out.AGI = m
	case AgentFI:
		out.FI = m
	case AgentCB:
		out.CB = m
	case AgentDR:
		out.DR = m
	case AgentCI:
		out.CI = m
	case AgentEI:
		out.EI = m
	}
}

func (va *VideoAgent) getMetricAgent(at AgentType) Agent {
	va.mu.Lock()
	defer va.mu.Unlock()
	if a, ok := va.metricAgents[at]; ok {
		return a
	}
	a := NewMetricAgent(at)
	va.metricAgents[at] = a
	return a
}
