// Package report 生成可视化报告。
// 读取 Phase 3 的 matrix.json + profiles/*.json，输出 HTML / Markdown / Vega-Lite。
package report

import (
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"mirror/internal/analysis"
)

// ──────────────────────────────────────────────
// 读取聚合结果
// ──────────────────────────────────────────────

type reportData struct {
	Matrix   *analysis.RelationMatrix
	Profiles map[string]*analysis.BehaviorProfile
	Summary  *analysis.Summary
}

func loadReportData(taskDir string) (*reportData, error) {
	// matrix.json
	matrix := &analysis.RelationMatrix{}
	data, err := os.ReadFile(filepath.Join(taskDir, "matrix.json"))
	if err != nil {
		return nil, fmt.Errorf("read matrix.json: %w", err)
	}
	if err := json.Unmarshal(data, matrix); err != nil {
		return nil, fmt.Errorf("parse matrix.json: %w", err)
	}

	// profiles
	profiles := make(map[string]*analysis.BehaviorProfile)
	entries, _ := os.ReadDir(filepath.Join(taskDir, "profiles"))
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(taskDir, "profiles", e.Name()))
		if err != nil {
			continue
		}
		var p analysis.BehaviorProfile
		if err := json.Unmarshal(data, &p); err != nil {
			continue
		}
		profiles[p.Community] = &p
	}

	// summary.json
	summary := &analysis.Summary{}
	data, err = os.ReadFile(filepath.Join(taskDir, "summary.json"))
	if err == nil {
		json.Unmarshal(data, summary)
	}

	return &reportData{Matrix: matrix, Profiles: profiles, Summary: summary}, nil
}

// ──────────────────────────────────────────────
// HTML 报告
// ──────────────────────────────────────────────

// GenerateHTML 生成独立 HTML 报告，包含嵌入式 Vega-Lite 可视化
func GenerateHTML(taskDir, outputPath string) error {
	rd, err := loadReportData(taskDir)
	if err != nil {
		return err
	}

	htmlContent := buildHTML(rd)
	return os.WriteFile(outputPath, []byte(htmlContent), 0644)
}

func buildHTML(rd *reportData) string {
	// 矩阵数据转 JS 数组
	matrixJS := buildMatrixJS(rd.Matrix)
	profileCards := buildProfileCards(rd.Profiles)
	summaryHTML := buildSummaryHTML(rd.Summary)

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<title>二游社区跨群体互动分析报告</title>
<script src="https://cdn.jsdelivr.net/npm/vega@5"></script>
<script src="https://cdn.jsdelivr.net/npm/vega-lite@5"></script>
<script src="https://cdn.jsdelivr.net/npm/vega-embed@6"></script>
<style>
body { font-family: -apple-system, BlinkMacSystemFont, sans-serif; max-width: 1200px; margin: 0 auto; padding: 20px; background: #f8f9fa; color: #333; }
h1 { color: #1a1a2e; border-bottom: 2px solid #e94560; padding-bottom: 10px; }
h2 { color: #16213e; margin-top: 40px; }
.card { background: white; border-radius: 8px; padding: 20px; margin: 16px 0; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
.card h3 { margin-top: 0; color: #e94560; }
.metric { display: inline-block; margin: 8px 16px 8px 0; padding: 8px 16px; background: #eef; border-radius: 20px; font-size: 14px; }
.metric .label { color: #666; }
.metric .value { font-weight: bold; color: #16213e; }
.chart-container { background: white; border-radius: 8px; padding: 20px; margin: 16px 0; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
table { width: 100%%; border-collapse: collapse; margin: 12px 0; }
th, td { padding: 8px 12px; text-align: left; border-bottom: 1px solid #dee2e6; }
th { background: #16213e; color: white; }
tr:hover { background: #f1f3f5; }
.bar { height: 20px; border-radius: 4px; display: inline-block; }
.bar-agi { background: #e94560; }
.bar-fi { background: #4ecdc4; }
.legend { display: flex; gap: 20px; margin: 12px 0; }
.legend-item { display: flex; align-items: center; gap: 6px; font-size: 14px; }
.legend-color { width: 16px; height: 16px; border-radius: 4px; }
</style>
</head>
<body>
<h1>📊 二游社区跨群体互动分析报告</h1>
%s

<h2>📈 社区关系热力图</h2>
<div class="legend">
  <div class="legend-item"><div class="legend-color" style="background:#e94560;"></div> 攻击性指数 (AGI)</div>
  <div class="legend-item"><div class="legend-color" style="background:#4ecdc4;"></div> 友善指数 (FI)</div>
</div>
<div id="heatmap" class="chart-container"></div>

<h2>🏘️ 群体行为画像</h2>
%s

<h2>📋 汇总概览</h2>
<div class="card">%s</div>

<script>
// 矩阵数据
const matrixData = %s;

// ── 热力图：攻击性指数 ──
const heatmapSpec = {
  "$schema": "https://vega.github.io/schema/vega-lite/v5.json",
  "title": "社区对平均攻击性指数 (AGI)",
  "data": { "values": matrixData },
  "mark": { "type": "rect", "tooltip": true },
  "encoding": {
    "x": { "field": "source", "type": "nominal", "title": "来源社区", "sort": null },
    "y": { "field": "target", "type": "nominal", "title": "目标社区", "sort": null },
    "color": {
      "field": "agi", "type": "quantitative",
      "title": "AGI",
      "scale": { "scheme": "reds", "domain": [0, 1] }
    },
    "tooltip": [
      { "field": "source", "type": "nominal", "title": "来源" },
      { "field": "target", "type": "nominal", "title": "目标" },
      { "field": "agi", "type": "quantitative", "title": "AGI", "format": ".3f" },
      { "field": "fi", "type": "quantitative", "title": "FI", "format": ".3f" }
    ]
  },
  "width": 400,
  "height": 400
};
vegaEmbed('#heatmap', heatmapSpec, { actions: false });
</script>
</body>
</html>`, summaryHTML, profileCards, summaryHTML, matrixJS)
}

// ──────────────────────────────────────────────
// 矩阵 → JS 数组
// ──────────────────────────────────────────────

func buildMatrixJS(m *analysis.RelationMatrix) string {
	var rows []string
	for _, src := range m.Communities {
		for _, tgt := range m.Communities {
			cell := m.Cells[src][tgt]
			agi := 0.0
			fi := 0.0
			if cell != nil {
				agi = cell.AvgAGI
				fi = cell.AvgFI
			}
			rows = append(rows, fmt.Sprintf(
				`{"source":"%s","target":"%s","agi":%.4f,"fi":%.4f}`,
				src, tgt, agi, fi,
			))
		}
	}
	return "[" + strings.Join(rows, ",\n") + "]"
}

// ──────────────────────────────────────────────
// 画像卡片
// ──────────────────────────────────────────────

func buildProfileCards(profiles map[string]*analysis.BehaviorProfile) string {
	// 按攻击性排序
	ordered := make([]*analysis.BehaviorProfile, 0, len(profiles))
	for _, p := range profiles {
		ordered = append(ordered, p)
	}
	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].AggressionIndex > ordered[j].AggressionIndex
	})

	var cards []string
	for _, p := range ordered {
		stars := ratingStars(p.AggressionIndex, 0.5)
		destRows := ""
		for _, d := range p.TopDestinations {
			destRows += fmt.Sprintf(`<tr><td>%s</td><td>%d</td><td>%.3f</td><td>%.3f</td></tr>`,
				html.EscapeString(d.Community), int(d.VisitRate*100), d.AvgAGI, d.AvgFI)
		}

		iaRows := ""
		for target, ratio := range p.InvasionAsymmetry {
			iaRows += fmt.Sprintf(`<tr><td>→ %s</td><td>%.2f</td></tr>`,
				html.EscapeString(target), ratio)
		}

		card := fmt.Sprintf(`<div class="card">
<h3>🏷️ %s</h3>
<div>
  <span class="metric"><span class="label">攻击性 </span><span class="value">%.3f %s</span></span>
  <span class="metric"><span class="label">友善度 </span><span class="value">%.3f</span></span>
  <span class="metric"><span class="label">拉踩倾向 </span><span class="value">%+.3f</span></span>
  <span class="metric"><span class="label">情绪烈度 </span><span class="value">%.3f</span></span>
  <span class="metric"><span class="label">串门率 </span><span class="value">%.1f%%</span></span>
</div>
<table><tr><th>去向</th><th>占比</th><th>AGI</th><th>FI</th></tr>%s</table>
%s
</div>`,
			html.EscapeString(p.Community),
			p.AggressionIndex, stars,
			p.FriendlinessIndex,
			p.ComparisonBias,
			p.EmotionalIntensity,
			p.VisitRate*100,
			destRows,
			buildIATable(iaRows),
		)
		cards = append(cards, card)
	}
	return strings.Join(cards, "\n")
}

func buildIATable(iaRows string) string {
	if iaRows == "" {
		return ""
	}
	return fmt.Sprintf(`<details><summary>入侵不对称性 (IA)</summary><table><tr><th>方向</th><th>A→B / B→A</th></tr>%s</table></details>`, iaRows)
}

func ratingStars(val float64, step float64) string {
	n := int(val / step)
	if n > 5 {
		n = 5
	}
	return strings.Repeat("⬆", n) + strings.Repeat("⬇", 5-n)
}

// ──────────────────────────────────────────────
// 概览
// ──────────────────────────────────────────────

func buildSummaryHTML(s *analysis.Summary) string {
	if s == nil {
		return "<p>无汇总数据</p>"
	}
	top3 := ""
	for _, m := range s.Top3Aggressive {
		for pair, val := range m {
			top3 += fmt.Sprintf("<li>%s: %.3f</li>", html.EscapeString(pair), val)
		}
	}
	return fmt.Sprintf(`<table>
<tr><th>社区数</th><td>%d</td></tr>
<tr><th>社区对数</th><td>%d</td></tr>
<tr><th>最高攻击性</th><td>%s</td></tr>
<tr><th>最高友善度</th><td>%s</td></tr>
<tr><th>最强拉踩</th><td>%s</td></tr>
<tr><th>Top 3 攻击对</th><td><ol>%s</ol></td></tr>
</table>`,
		s.Communities, s.Pairs,
		s.MaxAGI, s.MaxFI, s.MaxCB,
		top3,
	)
}
