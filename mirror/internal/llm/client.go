// Package llm 提供统一的 LLM API 客户端。
// 支持 OpenAI、Anthropic 以及兼容 OpenAI 协议的本地/自托管端点。
package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ──────────────────────────────────────────────
// 请求/响应类型
// ──────────────────────────────────────────────

// Message 对话消息
type Message struct {
	Role    string `json:"role"` // system | user | assistant
	Content string `json:"content"`
}

// ChatRequest 通用的聊天补全请求
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`

	// JSON 模式（结构化输出）
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
}

// ResponseFormat JSON 模式配置
type ResponseFormat struct {
	Type string `json:"type"` // "json_object" 或 "json_schema"
	// json_schema 的详细定义（如果需要）
	JSONSchema *JSONSchema `json:"json_schema,omitempty"`
}

// JSONSchema JSON Schema 定义
type JSONSchema struct {
	Name   string      `json:"name"`
	Schema interface{} `json:"schema"`
	Strict bool        `json:"strict"`
}

// ChatResponse 通用的聊天补全响应
type ChatResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
	Usage *Usage    `json:"usage,omitempty"`
	Error *APIError `json:"error,omitempty"`
}

// Usage token 用量
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// APIError API 错误
type APIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error: %s (type=%s, code=%s)", e.Message, e.Type, e.Code)
}

// ──────────────────────────────────────────────
// 客户端
// ──────────────────────────────────────────────

// Client LLM API 客户端
type Client struct {
	provider   string // openai | anthropic | local
	apiKey     string
	model      string
	baseURL    string
	timeout    time.Duration
	httpClient *http.Client
}

// Config 客户端配置
type Config struct {
	Provider string // openai | anthropic | local
	APIKey   string
	Model    string
	BaseURL  string
	Timeout  time.Duration
}

// NewClient 创建 LLM 客户端
func NewClient(cfg Config) *Client {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 60 * time.Second
	}
	baseURL := cfg.BaseURL
	if baseURL == "" {
		switch strings.ToLower(cfg.Provider) {
		case "openai":
			baseURL = "https://api.openai.com/v1"
		case "anthropic":
			// Anthropic 兼容 OpenAI 协议的代理通常用这个路径
			baseURL = "https://api.anthropic.com/v1"
		}
	}
	// 确保 baseURL 没有尾随斜杠
	baseURL = strings.TrimRight(baseURL, "/")

	return &Client{
		provider: cfg.Provider,
		apiKey:   cfg.APIKey,
		model:    cfg.Model,
		baseURL:  baseURL,
		timeout:  cfg.Timeout,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// Chat 发送聊天补全请求，返回解析后的响应
func (c *Client) Chat(req ChatRequest) (*ChatResponse, error) {
	if req.Model == "" {
		req.Model = c.model
	}
	if req.MaxTokens <= 0 {
		req.MaxTokens = 1024
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/chat/completions", c.baseURL)
	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	// Anthropic 需要额外的 header
	if strings.ToLower(c.provider) == "anthropic" {
		httpReq.Header.Set("anthropic-version", "2023-06-01")
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w (body: %s)", err, string(respBody))
	}

	if chatResp.Error != nil {
		return nil, chatResp.Error
	}

	return &chatResp, nil
}

// ChatJSON 发送请求并自动解析到目标结构体（JSON mode 快捷方法）
func (c *Client) ChatJSON(req ChatRequest, target interface{}) (*Usage, error) {
	// 启用 JSON 模式
	req.ResponseFormat = &ResponseFormat{
		Type: "json_object",
	}

	resp, err := c.Chat(req)
	if err != nil {
		return nil, err
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("empty response (no choices)")
	}

	content := resp.Choices[0].Message.Content
	if err := json.Unmarshal([]byte(content), target); err != nil {
		return nil, fmt.Errorf("parse JSON response: %w\nraw: %s", err, content)
	}

	return resp.Usage, nil
}

// Provider 返回客户端配置的 provider 名称
func (c *Client) Provider() string { return c.provider }

// Model 返回当前使用的模型名
func (c *Client) Model() string { return c.model }
