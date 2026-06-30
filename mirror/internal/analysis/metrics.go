// Package analysis 实现 Phase 3 量化指标引擎。
// 读取 Phase 2 产出的 JSON 分析结果，聚合计算跨群体互动指标，
// 输出社区关系矩阵和群体行为画像。
package analysis

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"mirror/internal/store"
)

// ──────────────────────────────────────────────
// 读取 Phase 2 的输出
// ──────────────────────────────────────────────

// LoadVideoOutputs 从任务目录读取所有视频分析结果
func LoadVideoOutputs(taskDir string) ([]*store.VideoOutput, error) {
	videosDir := filepath.Join(taskDir, "videos")
	entries, err := os.ReadDir(videosDir)
	if err != nil {
		return nil, fmt.Errorf("read videos dir: %w", err)
	}

	var outputs []*store.VideoOutput
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(videosDir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", e.Name(), err)
		}
		var vo store.VideoOutput
		if err := json.Unmarshal(data, &vo); err != nil {
			return nil, fmt.Errorf("parse %s: %w", e.Name(), err)
		}
		outputs = append(outputs, &vo)
	}

	sort.Slice(outputs, func(i, j int) bool {
		return outputs[i].Bvid < outputs[j].Bvid
	})
	return outputs, nil
}

// ──────────────────────────────────────────────
// 社区对指标聚合
// ──────────────────────────────────────────────

// PairKey (source, target) 社区对
type PairKey struct {
	Source string
	Target string
}

// PairMetrics 一个社区对的聚合指标
type PairMetrics struct {
	SourceCommunity string `json:"source_community"`
	TargetCommunity string `json:"target_community"`

	TotalComments int `json:"total_comments"`
	CrossComments int `json:"cross_comments"`

	// 平均分
	AvgVR  float64 `json:"avg_vr"`
	AvgAGI float64 `json:"avg_agi"`
	AvgFI  float64 `json:"avg_fi"`
	AvgCB  float64 `json:"avg_cb"`
	AvgDR  float64 `json:"avg_dr"`
	AvgCI  float64 `json:"avg_ci"`
	AvgEI  float64 `json:"avg_ei"`
}

// AggregatePairs 将所有评论按 (source, target) 分组聚合
func AggregatePairs(outputs []*store.VideoOutput) map[PairKey]*PairMetrics {
	// 先按 PairKey 收集所有分数
	type acc struct {
		count                                            int
		vrSum, agiSum, fiSum, cbSum, drSum, ciSum, eiSum float64
	}
	group := make(map[PairKey]*acc)

	allComments := 0
	for _, vo := range outputs {
		for _, c := range vo.Comments {
			allComments++
			if !c.IsCrossCommunity {
				continue
			}
			key := PairKey{Source: c.InferredCommunity, Target: c.AffiliatedCommunity}
			if key.Source == "" || key.Target == "" {
				continue
			}
			a, ok := group[key]
			if !ok {
				a = &acc{}
				group[key] = a
			}
			a.count++
			a.vrSum += scoreOrZero(c.VR)
			a.agiSum += scoreOrZero(c.AGI)
			a.fiSum += scoreOrZero(c.FI)
			a.cbSum += scoreOrZero(c.CB)
			a.drSum += scoreOrZero(c.DR)
			a.ciSum += scoreOrZero(c.CI)
			a.eiSum += scoreOrZero(c.EI)
		}
	}

	// 转为 PairMetrics
	result := make(map[PairKey]*PairMetrics)
	for key, a := range group {
		n := float64(a.count)
		result[key] = &PairMetrics{
			SourceCommunity: key.Source,
			TargetCommunity: key.Target,
			TotalComments:   allComments,
			CrossComments:   a.count,
			AvgVR:           a.vrSum / n,
			AvgAGI:          a.agiSum / n,
			AvgFI:           a.fiSum / n,
			AvgCB:           a.cbSum / n,
			AvgDR:           a.drSum / n,
			AvgCI:           a.ciSum / n,
			AvgEI:           a.eiSum / n,
		}
	}
	return result
}

func scoreOrZero(m *store.MetricOutput) float64 {
	if m == nil {
		return 0
	}
	return m.Score
}

// ──────────────────────────────────────────────
// 社区关系矩阵
// ──────────────────────────────────────────────

// RelationMatrix n×n 社区关系矩阵
type RelationMatrix struct {
	Communities []string                           `json:"communities"`
	Cells       map[string]map[string]*PairMetrics `json:"cells"` // [source][target]
}

// BuildMatrix 从聚合结果构建 n×n 矩阵
func BuildMatrix(pairs map[PairKey]*PairMetrics) *RelationMatrix {
	// 收集所有社区
	commSet := make(map[string]bool)
	for key := range pairs {
		commSet[key.Source] = true
		commSet[key.Target] = true
	}
	communities := make([]string, 0, len(commSet))
	for c := range commSet {
		communities = append(communities, c)
	}
	sort.Strings(communities)

	matrix := &RelationMatrix{
		Communities: communities,
		Cells:       make(map[string]map[string]*PairMetrics),
	}
	for _, s := range communities {
		row := make(map[string]*PairMetrics)
		for _, t := range communities {
			if s == t {
				row[t] = nil // 对角线的自我行为可不分析
			} else {
				row[t] = pairs[PairKey{Source: s, Target: t}]
			}
		}
		matrix.Cells[s] = row
	}
	return matrix
}

// ToJSON 输出 JSON 字节
func (m *RelationMatrix) ToJSON() ([]byte, error) {
	return json.MarshalIndent(m, "", "  ")
}
