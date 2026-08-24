package diagnosis

import (
	"sort"

	"task210-clocktrace/internal/model"
	"task210-clocktrace/internal/propagation"
)

// jumpToMillis 将纳秒跳变转为毫秒（评分基准单位）。
func jumpToMillis(ns int64) float64 {
	return float64(ns) / 1e6
}

// scoreSourceSwitch 计算源切换候选分数：跳变幅度 + 受影响节点数。
func scoreSourceSwitch(jump int64, res *propagation.PropagationResult) float64 {
	return jumpToMillis(jump) + float64(len(res.AnomalousNodes))
}

// scoreUpstreamJump 计算上游跳变候选分数：跳变幅度 + 受影响节点加权 + 偏斜度。
func scoreUpstreamJump(ns *propagation.NodeSeries, affected int) float64 {
	jump := int64(0)
	skew := 0.0
	if ns != nil {
		jump = ns.MaxJump
		skew = ns.Skew
	}
	return jumpToMillis(jump) + float64(affected)*0.5 + skew*10
}

// scoreLinkAnomaly 计算链路异常候选分数：受影响节点数 + 跳变幅度加权。
func scoreLinkAnomaly(res *propagation.PropagationResult) float64 {
	jump := int64(0)
	if ns, ok := res.Series[res.EarliestNode]; ok {
		jump = ns.MaxJump
	}
	return float64(len(res.AnomalousNodes)) + jumpToMillis(jump)*0.5
}

// RankCandidates 对候选按分数降序排序（稳定）。
func RankCandidates(cands []*model.RootCauseCandidate) []*model.RootCauseCandidate {
	out := append([]*model.RootCauseCandidate(nil), cands...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].ID < out[j].ID
	})
	return out
}
