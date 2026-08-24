package propagation

import (
	"task210-clocktrace/internal/timesync"
)

// SourceSwitchAnalysis 源切换分析结论。
type SourceSwitchAnalysis struct {
	Switch        SourceSwitch
	CausedJump    bool    // 源切换是否伴随显著偏移跳变
	JumpMagnitude int64   // 切换前后的偏移差（绝对值）
	Stability     float64 // 切换前后偏移序列的艾伦方差（稳定性）
}

// AnalyzeSourceSwitches 对已识别的源切换做归因分析：
// 判断切换是否伴随跳变，并量化切换前后的时钟稳定性变化。
func AnalyzeSourceSwitches(switches []SourceSwitch, series map[string]*NodeSeries) []SourceSwitchAnalysis {
	var out []SourceSwitchAnalysis
	for _, sw := range switches {
		a := SourceSwitchAnalysis{Switch: sw}
		if sw.OffsetDelta < 0 {
			a.JumpMagnitude = -sw.OffsetDelta
		} else {
			a.JumpMagnitude = sw.OffsetDelta
		}
		a.CausedJump = a.JumpMagnitude >= DefaultJumpThresholdNs

		if ns, ok := series[sw.NodeKey]; ok {
			a.Stability = allanDeviationOf(ns)
		}
		out = append(out, a)
	}
	return out
}

// allanDeviationOf 由偏移序列估算归一化频率偏差的艾伦方差。
func allanDeviationOf(ns *NodeSeries) float64 {
	if len(ns.Offsets) < 2 {
		return 0
	}
	freqs := make([]float64, 0, len(ns.Offsets)-1)
	for i := 1; i < len(ns.Offsets); i++ {
		// 相邻采样间偏移差即频率偏差（无量纲，相对采样间隔）。
		freqs = append(freqs, float64(ns.Offsets[i]-ns.Offsets[i-1]))
	}
	return timesync.AllanDeviation(freqs)
}
