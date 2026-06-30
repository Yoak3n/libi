package analysis

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
)

// AggregateResult Phase 3 的顶层输出
type AggregateResult struct {
	Matrix   *RelationMatrix             `json:"matrix"`
	Profiles map[string]*BehaviorProfile `json:"profiles"`
}

// RunAggregation 对指定任务目录运行完整聚合
// 输入: data/analyses/<task>/
// 输出: data/analyses/<task>/matrix.json
//
//	data/analyses/<task>/profiles/*.json
func RunAggregation(taskDir string) (*AggregateResult, error) {
	log.Printf("[aggregate] loading video outputs from %s", taskDir)

	// 1. 读取所有视频分析结果
	outputs, err := LoadVideoOutputs(taskDir)
	if err != nil {
		return nil, fmt.Errorf("load video outputs: %w", err)
	}
	log.Printf("[aggregate] loaded %d videos", len(outputs))

	if len(outputs) == 0 {
		return nil, fmt.Errorf("no video results found in %s/videos", taskDir)
	}

	// 2. 按社区对聚合
	pairs := AggregatePairs(outputs)
	log.Printf("[aggregate] aggregated %d community pairs", len(pairs))

	// 3. 构建关系矩阵
	matrix := BuildMatrix(pairs)

	// 4. 构建群体行为画像
	profiles := BuildProfiles(pairs)

	// 5. 补充总评论数
	totalComments := 0
	for _, vo := range outputs {
		for range vo.Comments {
			totalComments++
		}
	}
	for _, p := range profiles {
		p.TotalComments = totalComments
	}

	result := &AggregateResult{Matrix: matrix, Profiles: profiles}

	// 6. 写文件
	if err := writeResults(taskDir, result); err != nil {
		return nil, fmt.Errorf("write results: %w", err)
	}

	log.Printf("[aggregate] complete → %s/matrix.json + profiles/", taskDir)
	return result, nil
}

func writeResults(taskDir string, result *AggregateResult) error {
	// matrix.json
	matrixData, err := result.Matrix.ToJSON()
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(taskDir, "matrix.json"), matrixData, 0644); err != nil {
		return err
	}

	// profiles/
	profilesDir := filepath.Join(taskDir, "profiles")
	os.MkdirAll(profilesDir, 0755)
	for name, p := range result.Profiles {
		data, err := p.ToJSON()
		if err != nil {
			return fmt.Errorf("marshal profile %s: %w", name, err)
		}
		if err := os.WriteFile(filepath.Join(profilesDir, name+".json"), data, 0644); err != nil {
			return fmt.Errorf("write profile %s: %w", name, err)
		}
	}

	// summary.json — 汇总概览，方便快速查阅
	summary := buildSummary(result)
	summaryData, _ := json.MarshalIndent(summary, "", "  ")
	os.WriteFile(filepath.Join(taskDir, "summary.json"), summaryData, 0644)

	return nil
}

// Summary 概览
type Summary struct {
	Videos         int                  `json:"videos"`
	Communities    int                  `json:"communities"`
	Pairs          int                  `json:"community_pairs"`
	MaxAGI         string               `json:"highest_aggression"`
	MaxFI          string               `json:"highest_friendliness"`
	MaxCB          string               `json:"most_biased_comparison"`
	Top3Aggressive []map[string]float64 `json:"top_3_aggressive_pairs"`
}

func buildSummary(result *AggregateResult) *Summary {
	s := &Summary{
		Communities: len(result.Matrix.Communities),
		Pairs:       len(result.Profiles),
	}

	var maxAGI, maxFI, maxCB float64
	var maxAGIComm, maxFIComm, maxCBComm string
	type pairScore struct {
		Src string  `json:"source"`
		Tgt string  `json:"target"`
		AGI float64 `json:"agi"`
	}
	var topAGI []pairScore

	for _, pm := range result.Matrix.Cells {
		for _, m := range pm {
			if m == nil {
				continue
			}
			s.Videos += m.TotalComments / 100 // approximate
			if m.AvgAGI > maxAGI {
				maxAGI = m.AvgAGI
				maxAGIComm = fmt.Sprintf("%s→%s", m.SourceCommunity, m.TargetCommunity)
			}
			if m.AvgFI > maxFI {
				maxFI = m.AvgFI
				maxFIComm = fmt.Sprintf("%s→%s", m.SourceCommunity, m.TargetCommunity)
			}
			if m.AvgCB > maxCB {
				maxCB = m.AvgCB
				maxCBComm = fmt.Sprintf("%s→%s", m.SourceCommunity, m.TargetCommunity)
			}
			topAGI = append(topAGI, pairScore{Src: m.SourceCommunity, Tgt: m.TargetCommunity, AGI: m.AvgAGI})
		}
	}

	s.MaxAGI = maxAGIComm
	s.MaxFI = maxFIComm
	s.MaxCB = maxCBComm

	// Top 3 aggressive pairs
	sort.Slice(topAGI, func(i, j int) bool { return topAGI[i].AGI > topAGI[j].AGI })
	for i := 0; i < 3 && i < len(topAGI); i++ {
		s.Top3Aggressive = append(s.Top3Aggressive, map[string]float64{
			topAGI[i].Src + "→" + topAGI[i].Tgt: topAGI[i].AGI,
		})
	}

	return s
}
