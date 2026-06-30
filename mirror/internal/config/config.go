// Package config 提供 mirror 的配置管理。
//
// 文件隔离：mirror 读 mirror.yaml，不与 troll/shared 的 config.yaml 冲突。
// 敏感隔离：LLM API Key 从环境变量读取，不写入配置文件。
//
// 环境变量（按优先级）：
//
//	MIRROR_LLM_API_KEY        通用 LLM API Key
//	OPENAI_API_KEY            provider=openai 时读取
//	ANTHROPIC_API_KEY         provider=anthropic 时读取
//	MIRROR_LLM_BASE_URL       LLM base URL（覆盖配置文件中的值）
package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
	"go.yaml.in/yaml/v3"
)

// ──────────────────────────────────────────────
// 配置类型定义
// ──────────────────────────────────────────────

// LLMConfig 大语言模型配置（不含 API Key——从环境变量读取）
type LLMConfig struct {
	Provider string `yaml:"provider"` // openai | anthropic | local
	Model    string `yaml:"model"`    // 模型名，如 gpt-4o, claude-sonnet-4
	BaseURL  string `yaml:"base_url"` // 可选：自定义 API 地址
	Timeout  int    `yaml:"timeout"`  // 请求超时秒数
}

// AgentConfig Agent 调度配置
type AgentConfig struct {
	MaxRetries    int `yaml:"max_retries"`    // 单个 Agent 最大重试次数
	ParallelCount int `yaml:"parallel_count"` // 并行 Agent 数量
	TimeoutSec    int `yaml:"timeout_sec"`    // Agent 超时秒数
}

// CommunityDef 社区定义（用于 mirror.yaml 中的 communities 节）
type CommunityDef struct {
	Name    string   `yaml:"name"`    // 社区名称（如 原神）
	Aliases []string `yaml:"aliases"` // 别名列表（如 原神, Genshin, 原）
	Tags    []string `yaml:"tags"`    // 关联标签（如 开放世界, 米哈游）
	Weight  int      `yaml:"weight"`  // 优先级权重（社区归属判定时使用）
}

// TrollRef 对 troll 配置的引用
// mirror 可以复用在 troll 中已配置好的 Bilibili cookie/数据库等
type TrollRef struct {
	ConfigPath string `yaml:"config_path"` // troll 的 config.yaml 路径（可选）
	DataDir    string `yaml:"data_dir"`    // troll 的数据目录（可选）
}

// MirrorConfig mirror 顶层配置
type MirrorConfig struct {
	// 工作模式
	WorkDir string `yaml:"work_dir"` // 工作目录（数据输出路径）

	// Troll 引用（用于复用 Bilibili 认证和数据库）
	Troll *TrollRef `yaml:"troll"`

	// 社区定义（仅名称/别名映射，实际数据存数据库）
	Communities []CommunityDef `yaml:"communities"`

	// LLM 配置
	LLM LLMConfig `yaml:"llm"`

	// Agent 调度配置
	Agent AgentConfig `yaml:"agent"`

	// 分析阈值
	CommunityConfidenceThreshold float64 `yaml:"community_confidence_threshold"`
	MaxCommentsPerVideo          int     `yaml:"max_comments_per_video"`
}

// ──────────────────────────────────────────────
// 全局变量
// ──────────────────────────────────────────────

// Conf 全局配置实例（通过 Init() 初始化）
var Conf *MirrorConfig
var v *viper.Viper

// ──────────────────────────────────────────────
// 初始化
// ──────────────────────────────────────────────

// Init 初始化 mirror 配置。
// 从 mirror.yaml 读取，不会触碰 config.yaml，避免与 troll/shared 冲突。
func Init() {
	v = viper.New()
	v.SetConfigName("mirror")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")

	// 默认值
	setDefaults()

	// 尝试读取 mirror.yaml，不存在时使用默认值
	_ = v.ReadInConfig()

	Conf = &MirrorConfig{}
	if err := v.Unmarshal(Conf); err != nil {
		panic("failed to parse mirror config: " + err.Error())
	}

	// 补充默认值（Unmarshal 不会填充零值字段）
	fillDefaults(Conf)
}

func setDefaults() {
	v.SetDefault("work_dir", "data")
	v.SetDefault("llm.provider", "openai")
	v.SetDefault("llm.model", "gpt-4o")
	v.SetDefault("llm.timeout", 60)
	v.SetDefault("agent.max_retries", 3)
	v.SetDefault("agent.parallel_count", 5)
	v.SetDefault("agent.timeout_sec", 120)
	v.SetDefault("community_confidence_threshold", 0.6)
	v.SetDefault("max_comments_per_video", 0)
}

func fillDefaults(c *MirrorConfig) {
	if c.WorkDir == "" {
		c.WorkDir = "data"
	}
	if c.LLM.Timeout <= 0 {
		c.LLM.Timeout = 60
	}
	if c.Agent.MaxRetries <= 0 {
		c.Agent.MaxRetries = 3
	}
	if c.Agent.ParallelCount <= 0 {
		c.Agent.ParallelCount = 5
	}
	if c.Agent.TimeoutSec <= 0 {
		c.Agent.TimeoutSec = 120
	}
	if c.CommunityConfidenceThreshold <= 0 {
		c.CommunityConfidenceThreshold = 0.6
	}
}

// ──────────────────────────────────────────────
// LLM API Key 解析
// ──────────────────────────────────────────────

// ResolveAPIKey 返回当前配置的 LLM API Key。
// 按优先级依次尝试：OPENAI_API_KEY → ANTHROPIC_API_KEY → MIRROR_LLM_API_KEY
// 不依赖 provider 配置值，所有 Key 都试一遍。
func ResolveAPIKey() string {
	if Conf == nil {
		return ""
	}
	// 全量尝试，不依赖 provider 值
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		return key
	}
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		return key
	}
	if key := os.Getenv("MIRROR_LLM_API_KEY"); key != "" {
		return key
	}
	return ""
}

// ResolveBaseURL 返回 LLM base URL。
// 环境变量 MIRROR_LLM_BASE_URL 可覆盖配置文件中的值。
func ResolveBaseURL() string {
	if env := os.Getenv("MIRROR_LLM_BASE_URL"); env != "" {
		return env
	}
	if Conf != nil {
		return Conf.LLM.BaseURL
	}
	return ""
}

// HasValidLLM 检查 LLM 配置是否可用（有 Provider + 有 Key）
func HasValidLLM() bool {
	if Conf == nil {
		return false
	}
	return Conf.LLM.Provider != "" && ResolveAPIKey() != ""
}

// ──────────────────────────────────────────────
// 配置持久化
// ──────────────────────────────────────────────

// Save 将当前内存中的配置写回 mirror.yaml。
// 注意：不保存 API Key——它只存在于环境变量中。
func Save() error {
	if v == nil {
		return fmt.Errorf("config not initialized")
	}
	data, err := yaml.Marshal(Conf)
	if err != nil {
		return err
	}
	configFile := v.ConfigFileUsed()
	if configFile == "" {
		configFile = "mirror.yaml"
	}
	return os.WriteFile(configFile, data, 0644)
}

// ──────────────────────────────────────────────
// 社区查找
// ──────────────────────────────────────────────

// GetCommunityNames 返回所有已定义的社区名称列表
func GetCommunityNames() []string {
	if Conf == nil {
		return nil
	}
	names := make([]string, len(Conf.Communities))
	for i, c := range Conf.Communities {
		names[i] = c.Name
	}
	return names
}

// FindCommunity 根据名称或别名查找社区定义
func FindCommunity(name string) *CommunityDef {
	if Conf == nil {
		return nil
	}
	name = strings.TrimSpace(name)
	for i := range Conf.Communities {
		c := &Conf.Communities[i]
		if strings.EqualFold(c.Name, name) {
			return c
		}
		for _, alias := range c.Aliases {
			if strings.EqualFold(alias, name) {
				return c
			}
		}
	}
	return nil
}
