// Package report — 趋势分析 + 时序派生指标（Δ-AGI、关系升温/降温、节奏事件冲击、稳态水平）
package report

import (
	"encoding/json"
	"fmt"
	"html"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"mirror/internal/analysis"
)

// ──────────────────────────────────────────────
// 趋势数据
// ──────────────────────────────────────────────

// TrendPoint 一个时段的一个指标值
type TrendPoint struct {
	TaskName  string  `json:"task_name"`
	Timestamp string  `json:"timestamp"`
	AGI       float64 `json:"agi"`
	FI        float64 `json:"fi"`
	CB        float64 `json:"cb"`
	EI        float64 `json:"ei"`
	VR        float64 `json:"vr"`
}

// DerivedMetrics 时序派生指标
type DerivedMetrics struct {
	DeltaAGI      []float64 `json:"delta_agi"`      // 相邻时段 AGI 变化量
	DeltaFI       []float64 `json:"delta_fi"`       // 相邻时段 FI 变化量
	AGIVolatility float64   `json:"agi_volatility"` // AGI 标准差（节奏事件冲击代理）
	FIVolatility  float64   `json:"fi_volatility"`  // FI 标准差
	SteadyAGI     float64   `json:"steady_agi"`     // AGI 稳态水平（中位数）
	SteadyFI      float64   `json:"steady_fi"`      // FI 稳态水平
	TrendAGI      string    `json:"trend_agi"`      // AGI 趋势方向：上升/下降/震荡/持平
	TrendFI       string    `json:"trend_fi"`       // FI 趋势方向
	MaxDeltaAGI   float64   `json:"max_delta_agi"`  // 最大单次 AGI 变化
	MaxDeltaFI    float64   `json:"max_delta_fi"`   // 最大单次 FI 变化
	WarmingLabel  string    `json:"warming_label"`  // 关系升温/降温描述
}

// TrendSeries 一个社区对的时序数据 + 派生指标
type TrendSeries struct {
	Source  string          `json:"source"`
	Target  string          `json:"target"`
	Label   string          `json:"label"`
	Points  []TrendPoint    `json:"points"`
	Derived *DerivedMetrics `json:"derived,omitempty"`
}

// TrendResult 完整趋势结果
type TrendResult struct {
	SeriesList []TrendSeries `json:"series"`
}

// ──────────────────────────────────────────────
// 加载多时段数据
// ──────────────────────────────────────────────

func LoadTrend(taskDirs []string) (*TrendResult, error) {
	if len(taskDirs) == 0 {
		return nil, fmt.Errorf("no task directories provided")
	}
	type key struct{ src, tgt string }
	seriesMap := make(map[key]*TrendSeries)

	for _, dir := range taskDirs {
		taskName := filepath.Base(dir)
		matrix := &analysis.RelationMatrix{}
		data, err := os.ReadFile(filepath.Join(dir, "matrix.json"))
		if err != nil {
			return nil, fmt.Errorf("read %s/matrix.json: %w", dir, err)
		}
		if err := json.Unmarshal(data, matrix); err != nil {
			return nil, fmt.Errorf("parse %s/matrix.json: %w", dir, err)
		}
		for _, src := range matrix.Communities {
			for _, tgt := range matrix.Communities {
				if src == tgt {
					continue
				}
				cell := matrix.Cells[src][tgt]
				k := key{src, tgt}
				pt := TrendPoint{TaskName: taskName, Timestamp: taskName}
				if cell != nil {
					pt.AGI = cell.AvgAGI
					pt.FI = cell.AvgFI
					pt.CB = cell.AvgCB
					pt.EI = cell.AvgEI
					pt.VR = safeDiv(float64(cell.CrossComments), float64(max(cell.TotalComments, 1)))
				}
				if _, ok := seriesMap[k]; !ok {
					seriesMap[k] = &TrendSeries{
						Source: src, Target: tgt,
						Label: fmt.Sprintf("%s → %s", src, tgt),
					}
				}
				seriesMap[k].Points = append(seriesMap[k].Points, pt)
			}
		}
	}

	result := &TrendResult{}
	for _, s := range seriesMap {
		computeDerived(s)
		result.SeriesList = append(result.SeriesList, *s)
	}
	sort.Slice(result.SeriesList, func(i, j int) bool {
		return result.SeriesList[i].Label < result.SeriesList[j].Label
	})
	return result, nil
}

// ──────────────────────────────────────────────
// 派生指标计算
// ──────────────────────────────────────────────

func computeDerived(s *TrendSeries) {
	n := len(s.Points)
	if n < 2 {
		return
	}

	d := &DerivedMetrics{}

	// Δ 差值
	var agiVals, fiVals []float64
	for i := 1; i < n; i++ {
		dagi := s.Points[i].AGI - s.Points[i-1].AGI
		dfi := s.Points[i].FI - s.Points[i-1].FI
		d.DeltaAGI = append(d.DeltaAGI, dagi)
		d.DeltaFI = append(d.DeltaFI, dfi)
		agiVals = append(agiVals, s.Points[i].AGI)
		fiVals = append(fiVals, s.Points[i].FI)
	}

	// 标准差（波动性 = 节奏事件冲击的代理）
	d.AGIVolatility = stdDev(agiVals)
	d.FIVolatility = stdDev(fiVals)

	// 稳态水平 = 中位数
	d.SteadyAGI = median(agiVals)
	d.SteadyFI = median(fiVals)

	// 最大单次变化
	for _, v := range d.DeltaAGI {
		if abs(v) > abs(d.MaxDeltaAGI) {
			d.MaxDeltaAGI = v
		}
	}
	for _, v := range d.DeltaFI {
		if abs(v) > abs(d.MaxDeltaFI) {
			d.MaxDeltaFI = v
		}
	}

	// 趋势方向判断：使用最后一段 vs 前半段的均值比较
	half := n / 2
	var earlyAGI, lateAGI, earlyFI, lateFI float64
	for i := 0; i < half; i++ {
		earlyAGI += s.Points[i].AGI
		earlyFI += s.Points[i].FI
	}
	for i := half; i < n; i++ {
		lateAGI += s.Points[i].AGI
		lateFI += s.Points[i].FI
	}
	earlyAGI /= float64(half)
	lateAGI /= float64(n - half)
	earlyFI /= float64(half)
	lateFI /= float64(n - half)

	d.TrendAGI = trendLabel(earlyAGI, lateAGI, d.AGIVolatility)
	d.TrendFI = trendLabel(earlyFI, lateFI, d.FIVolatility)

	// 关系升温/降温
	totalDeltaFI := s.Points[n-1].FI - s.Points[0].FI
	switch {
	case totalDeltaFI > 0.1:
		d.WarmingLabel = fmt.Sprintf("🔥 明显升温 (ΔFI=%.3f)", totalDeltaFI)
	case totalDeltaFI > 0.03:
		d.WarmingLabel = fmt.Sprintf("📈 轻微升温 (ΔFI=%.3f)", totalDeltaFI)
	case totalDeltaFI < -0.1:
		d.WarmingLabel = fmt.Sprintf("❄️ 明显降温 (ΔFI=%.3f)", totalDeltaFI)
	case totalDeltaFI < -0.03:
		d.WarmingLabel = fmt.Sprintf("📉 轻微降温 (ΔFI=%.3f)", totalDeltaFI)
	default:
		d.WarmingLabel = fmt.Sprintf("➖ 基本持平 (ΔFI=%.3f)", totalDeltaFI)
	}

	s.Derived = d
}

func trendLabel(early, late, vol float64) string {
	delta := late - early
	threshold := max(vol*0.5, 0.02) // 最小可感知变化
	switch {
	case delta > threshold:
		return "上升 ↑"
	case delta < -threshold:
		return "下降 ↓"
	case vol > 0.1:
		return "震荡 ~"
	default:
		return "持平 →"
	}
}

func stdDev(vals []float64) float64 {
	if len(vals) < 2 {
		return 0
	}
	mean := 0.0
	for _, v := range vals {
		mean += v
	}
	mean /= float64(len(vals))
	var sumSq float64
	for _, v := range vals {
		d := v - mean
		sumSq += d * d
	}
	return math.Sqrt(sumSq / float64(len(vals)))
}

func median(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sorted := make([]float64, len(vals))
	copy(sorted, vals)
	sort.Float64s(sorted)
	return sorted[len(sorted)/2]
}

func safeDiv(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

// ──────────────────────────────────────────────
// JSON 导出
// ──────────────────────────────────────────────

// GenerateTrendJSON 派生指标以 JSON 格式输出
func GenerateTrendJSON(taskDirs []string, outputPath string) error {
	trend, err := LoadTrend(taskDirs)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(trend, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, data, 0644)
}

// ──────────────────────────────────────────────
// HTML 趋势报告（含派生指标展示）
// ──────────────────────────────────────────────

func GenerateTrendHTML(taskDirs []string, outputPath string) error {
	trend, err := LoadTrend(taskDirs)
	if err != nil {
		return err
	}
	htmlContent := buildTrendHTML(trend, taskDirs)
	return os.WriteFile(outputPath, []byte(htmlContent), 0644)
}

func buildTrendHTML(trend *TrendResult, allDirs []string) string {
	seriesData := buildTrendSeriesJS(trend)
	dirList := buildDirList(allDirs)

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<title>跨群体互动趋势分析</title>
<script src="https://cdn.jsdelivr.net/npm/vega@5"></script>
<script src="https://cdn.jsdelivr.net/npm/vega-lite@5"></script>
<script src="https://cdn.jsdelivr.net/npm/vega-embed@6"></script>
<style>
body { font-family: -apple-system, BlinkMacSystemFont, sans-serif; max-width: 1200px; margin: 0 auto; padding: 20px; background: #f8f9fa; }
h1 { color: #1a1a2e; border-bottom: 2px solid #e94560; }
h2 { color: #16213e; margin-top: 40px; }
.chart { background: white; border-radius: 8px; padding: 20px; margin: 16px 0; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
.info { background: #eef; border-radius: 8px; padding: 12px 20px; margin: 16px 0; font-size: 14px; }
select { padding: 8px 12px; border-radius: 6px; border: 1px solid #ccc; font-size: 14px; margin-bottom: 16px; }
.derived-card { background: white; border-radius: 8px; padding: 20px; margin: 16px 0; box-shadow: 0 2px 4px rgba(0,0,0,0.1); display: none; }
.derived-card h3 { margin-top: 0; color: #e94560; }
.metric { display: inline-block; margin: 6px 12px 6px 0; padding: 6px 14px; background: #eef; border-radius: 16px; font-size: 13px; }
.metric .label { color: #666; }
.metric .value { font-weight: bold; }
.up { color: #e94560; }
.down { color: #4ecdc4; }
.shock { color: #f39c12; }
.flat { color: #95a5a6; }
table { width: 100%%; border-collapse: collapse; margin: 8px 0; font-size: 14px; }
th, td { padding: 6px 10px; text-align: left; border-bottom: 1px solid #dee2e6; }
th { background: #16213e; color: white; }
</style>
</head>
<body>
<h1>📈 跨群体互动趋势分析</h1>
<div class="info">
  <strong>分析时段：</strong>
  <ul>%s</ul>
</div>
<div>
  <label for="pairSelect"><strong>选择社区对：</strong></label>
  <select id="pairSelect" onchange="updateChart()"></select>
</div>
<div id="trendChart" class="chart"></div>
<div id="derivedCard" class="derived-card">
  <h3>📊 时序派生指标</h3>
  <div id="derivedContent"></div>
</div>

<script>
const trendData = %s;

function fillSelect() {
  const sel = document.getElementById('pairSelect');
  trendData.forEach((s, i) => {
    const opt = document.createElement('option');
    opt.value = i; opt.text = s.label;
    sel.appendChild(opt);
  });
}

function updateChart() {
  const idx = parseInt(document.getElementById('pairSelect').value);
  const s = trendData[idx];
  if (!s) return;

  // 折线图
  const vals = [];
  s.points.forEach(p => {
    vals.push({ time: p.timestamp, metric: 'AGI 攻击性', value: p.agi });
    vals.push({ time: p.timestamp, metric: 'FI 友善度', value: p.fi });
  });
  vegaEmbed('#trendChart', {
    "$schema": "https://vega.github.io/schema/vega-lite/v5.json",
    "title": s.label + " 趋势",
    "data": { "values": vals },
    "mark": { "type": "line", "point": { "size": 80 }, "strokeWidth": 2.5 },
    "encoding": {
      "x": { "field": "time", "type": "nominal", "title": "时段", "sort": null },
      "y": { "field": "value", "type": "quantitative", "title": "分数", "scale": { "domain": [0, 1] } },
      "color": { "field": "metric", "type": "nominal", "scale": { "range": ["#e94560", "#4ecdc4"] } }
    },
    "width": 700, "height": 400
  }, { actions: false });

  // 派生指标卡片
  const d = s.derived;
  if (!d) { document.getElementById('derivedCard').style.display = 'none'; return; }
  document.getElementById('derivedCard').style.display = 'block';

  const trendClass = d.trend_agi.indexOf('上升') >= 0 ? 'up' : d.trend_agi.indexOf('下降') >= 0 ? 'down' : d.trend_agi.indexOf('震荡') >= 0 ? 'shock' : 'flat';
  var html = '<div>' +
    '<span class="metric"><span class="label">AGI 趋势 </span><span class="value ' + trendClass + '">' + d.trend_agi + '</span></span>' +
    '<span class="metric"><span class="label">FI 趋势 </span><span class="value ' + trendClass + '">' + d.trend_fi + '</span></span>' +
    '<span class="metric"><span class="label">稳态 AGI </span><span class="value">' + d.steady_agi.toFixed(3) + '</span></span>' +
    '<span class="metric"><span class="label">稳态 FI </span><span class="value">' + d.steady_fi.toFixed(3) + '</span></span>' +
    '<span class="metric"><span class="label">AGI 波动 </span><span class="value">' + d.agi_volatility.toFixed(3) + '</span></span>' +
    '<span class="metric"><span class="label">FI 波动 </span><span class="value">' + d.fi_volatility.toFixed(3) + '</span></span>' +
    '<span class="metric"><span class="label">最大 ΔAGI </span><span class="value">' + (d.max_delta_agi >= 0 ? '+' : '') + d.max_delta_agi.toFixed(3) + '</span></span>' +
    '<span class="metric"><span class="label">最大 ΔFI </span><span class="value">' + (d.max_delta_fi >= 0 ? '+' : '') + d.max_delta_fi.toFixed(3) + '</span></span>' +
    '</div>' +
    '<div style="margin-top:12px"><strong>' + d.warming_label + '</strong></div>' +
    '<table style="margin-top:12px"><tr><th>时段</th><th>AGI</th><th>FI</th><th>Δ-AGI</th><th>Δ-FI</th></tr>' +
    s.points.map(function(p, i) { return '<tr><td>' + p.timestamp + '</td><td>' + p.agi.toFixed(3) + '</td><td>' + p.fi.toFixed(3) + '</td><td>' + (i > 0 ? (d.delta_agi[i-1] >= 0 ? '+' : '') + d.delta_agi[i-1].toFixed(3) : '-') + '</td><td>' + (i > 0 ? (d.delta_fi[i-1] >= 0 ? '+' : '') + d.delta_fi[i-1].toFixed(3) : '-') + '</td></tr>'; }).join('') +
    '</table>';
  document.getElementById('derivedContent').innerHTML = html;
}

fillSelect();
updateChart();
</script>
</body>
</html>`, dirList, seriesData)
}

func buildTrendSeriesJS(trend *TrendResult) string {
	b, _ := json.Marshal(trend.SeriesList)
	return string(b)
}

func buildDirList(dirs []string) string {
	var items []string
	for _, d := range dirs {
		items = append(items, fmt.Sprintf("<li>%s</li>", html.EscapeString(filepath.Base(d))))
	}
	return strings.Join(items, "\n")
}
