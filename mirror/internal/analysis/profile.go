package analysis

import (
	"encoding/json"
	"sort"
)

// ──────────────────────────────────────────────
// 群体行为画像
// ──────────────────────────────────────────────

// TopDestination 最爱串门的去向
type TopDestination struct {
	Community string  `json:"community"`
	VisitRate float64 `json:"visit_rate"` // 串门占比
	AvgAGI    float64 `json:"avg_agi"`
	AvgFI     float64 `json:"avg_fi"`
}

// BehaviorProfile 一个社区的完整行为画像
type BehaviorProfile struct {
	Community string `json:"community"`

	// 总体统计
	TotalComments int     `json:"total_comments"`
	CrossComments int     `json:"cross_comments"`
	VisitRate     float64 `json:"visit_rate"` // VR：串门率

	// 核心指标
	AggressionIndex     float64 `json:"aggression_index"`     // AGI
	FriendlinessIndex   float64 `json:"friendliness_index"`   // FI
	ComparisonBias      float64 `json:"comparison_bias"`      // CB
	DefensiveReactivity float64 `json:"defensive_reactivity"` // DR
	ConflictInvolvement float64 `json:"conflict_involvement"` // CI
	EmotionalIntensity  float64 `json:"emotional_intensity"`  // EI

	// 派生指标
	InvasionAsymmetry map[string]float64 `json:"invasion_asymmetry"` // IA: A→B / B→A

	// 串门排名
	TopDestinations []TopDestination `json:"top_destinations"`

	// 争议热点（从 controversy JSON 读取，外部注入）
	ControversyTopics []string `json:"controversy_topics,omitempty"`
}

// BuildProfiles 从 PairMetrics 和 RelationMatrix 构建所有社区的行为画像
func BuildProfiles(pairs map[PairKey]*PairMetrics) map[string]*BehaviorProfile {
	// 按 Source 社区分组
	type sourceGroup struct {
		totalCross                                int
		agiSum, fiSum, cbSum, drSum, ciSum, eiSum float64
		dests                                     []PairMetrics
	}
	groups := make(map[string]*sourceGroup)
	allCross := 0

	for _, pm := range pairs {
		allCross += pm.CrossComments
		g, ok := groups[pm.SourceCommunity]
		if !ok {
			g = &sourceGroup{}
			groups[pm.SourceCommunity] = g
		}
		g.totalCross += pm.CrossComments
		n := float64(pm.CrossComments)
		g.agiSum += pm.AvgAGI * n
		g.fiSum += pm.AvgFI * n
		g.cbSum += pm.AvgCB * n
		g.drSum += pm.AvgDR * n
		g.ciSum += pm.AvgCI * n
		g.eiSum += pm.AvgEI * n
		g.dests = append(g.dests, *pm)
	}

	profiles := make(map[string]*BehaviorProfile)
	for comm, g := range groups {
		nc := float64(g.totalCross)
		p := &BehaviorProfile{
			Community:           comm,
			TotalComments:       0, // 由外部填充
			CrossComments:       g.totalCross,
			VisitRate:           float64(g.totalCross) / float64(max(allCross, 1)),
			AggressionIndex:     g.agiSum / nc,
			FriendlinessIndex:   g.fiSum / nc,
			ComparisonBias:      g.cbSum / nc,
			DefensiveReactivity: g.drSum / nc,
			ConflictInvolvement: g.ciSum / nc,
			EmotionalIntensity:  g.eiSum / nc,
			InvasionAsymmetry:   make(map[string]float64),
			TopDestinations:     make([]TopDestination, 0),
		}

		// 串门去向排名
		for _, d := range g.dests {
			p.TopDestinations = append(p.TopDestinations, TopDestination{
				Community: d.TargetCommunity,
				VisitRate: float64(d.CrossComments) / nc,
				AvgAGI:    d.AvgAGI,
				AvgFI:     d.AvgFI,
			})
		}
		sort.Slice(p.TopDestinations, func(i, j int) bool {
			return p.TopDestinations[i].VisitRate > p.TopDestinations[j].VisitRate
		})

		profiles[comm] = p
	}

	// IA：入侵不对称性（需要双边都有数据才能算）
	for _, pm := range pairs {
		reverse := PairKey{Source: pm.TargetCommunity, Target: pm.SourceCommunity}
		rev, ok := pairs[reverse]
		if !ok || rev.CrossComments == 0 {
			continue
		}
		ia := float64(pm.CrossComments) / float64(rev.CrossComments)
		if p, ok := profiles[pm.SourceCommunity]; ok {
			p.InvasionAsymmetry[pm.TargetCommunity] = ia
		}
	}

	return profiles
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ToJSON 输出 JSON
func (p *BehaviorProfile) ToJSON() ([]byte, error) {
	return json.MarshalIndent(p, "", "  ")
}
