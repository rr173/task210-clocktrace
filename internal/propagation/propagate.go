// Package propagation 沿同步拓扑计算偏移传播、识别源切换与最早异常链路。
package propagation

import (
	"sort"

	"task210-clocktrace/internal/model"
	"task210-clocktrace/internal/store"
	"task210-clocktrace/internal/timesync"
	"task210-clocktrace/internal/topology"
)

// NodeSeries 单个节点的偏移采样序列与跳变统计。
type NodeSeries struct {
	NodeKey   string
	Offsets   []int64  // 按序号升序的偏移（纳秒）
	Sequences []int64  // 对应序号
	Sources   []string // 对应时钟源
	MaxJump   int64    // 最大相邻跳变幅度
	JumpAt    int      // 最大跳变发生的位置（Offsets 下标，0 表示无跳变）
	Skew      float64  // 与上游节点的相对偏斜度
}

// SourceSwitch 一次时钟源切换。
type SourceSwitch struct {
	NodeKey     string
	FromSource  string
	ToSource    string
	AtSequence  int64
	OffsetDelta int64
}

// PropagationResult 传播分析结果。
type PropagationResult struct {
	Series         map[string]*NodeSeries
	AnomalousNodes []string       // 跳变幅度达到阈值的节点
	EarliestNode   string         // 拓扑上最早出现异常跳变的节点
	SourceSwitches []SourceSwitch // 识别到的源切换
	MaxOffset      int64
	MaxRoundtrip   int64
}

// Manager 传播分析器。
type Manager struct {
	db *store.DB
}

// New 创建传播分析器。
func New(db *store.DB) *Manager { return &Manager{db: db} }

// Analyze 分析快照的偏移传播。
// jumpThresholdNs 为跳变判定阈值（纳秒）；小于等于 0 时使用默认 1ms。
func (m *Manager) Analyze(snapshotID int64, jumpThresholdNs int64) (*PropagationResult, error) {
	if jumpThresholdNs <= 0 {
		jumpThresholdNs = DefaultJumpThresholdNs
	}
	links, err := m.db.ListLinks(snapshotID)
	if err != nil {
		return nil, err
	}
	samples, err := m.db.ListSamplesBySnapshot(snapshotID)
	if err != nil {
		return nil, err
	}

	byNode := groupSamples(samples)
	res := &PropagationResult{Series: map[string]*NodeSeries{}}

	for nodeKey, ss := range byNode {
		sort.Slice(ss, func(i, j int) bool { return ss[i].Sequence < ss[j].Sequence })
		ns := &NodeSeries{NodeKey: nodeKey}
		for _, s := range ss {
			ns.Offsets = append(ns.Offsets, s.OffsetNs)
			ns.Sequences = append(ns.Sequences, s.Sequence)
			ns.Sources = append(ns.Sources, s.SourceID)
			if s.OffsetNs > res.MaxOffset {
				res.MaxOffset = s.OffsetNs
			}
			if s.RoundtripNs > res.MaxRoundtrip {
				res.MaxRoundtrip = s.RoundtripNs
			}
		}
		m.computeJumps(ns)
		m.computeSourceSwitches(ns, res)
		res.Series[nodeKey] = ns
	}

	// 识别异常节点。
	for nodeKey, ns := range res.Series {
		if ns.MaxJump > jumpThresholdNs {
			res.AnomalousNodes = append(res.AnomalousNodes, nodeKey)
		}
	}

	// 计算最早异常节点：异常节点中拓扑层级最上游者。
	res.EarliestNode = m.earliestAnomalous(links, res.AnomalousNodes)

	// 计算每个异常节点相对上游的偏斜度。
	g := topology.BuildGraph(links)
	for _, nodeKey := range res.AnomalousNodes {
		up := g.Upstream(nodeKey)
		if len(up) == 0 {
			continue
		}
		// 取第一个存在序列的上游节点。
		for _, u := range up {
			us, ok := res.Series[u]
			if !ok || len(us.Offsets) < 2 {
				continue
			}
			ds := res.Series[nodeKey]
			res.Series[nodeKey].Skew = timesync.RelativeSkew(us.Offsets, ds.Offsets)
			break
		}
	}
	return res, nil
}

func (m *Manager) computeJumps(ns *NodeSeries) {
	for i := 1; i < len(ns.Offsets); i++ {
		jump := timesync.JumpMagnitude(ns.Offsets[i-1], ns.Offsets[i])
		if jump > ns.MaxJump {
			ns.MaxJump = jump
			ns.JumpAt = i
		}
	}
}

func (m *Manager) computeSourceSwitches(ns *NodeSeries, res *PropagationResult) {
	for i := 1; i < len(ns.Sources); i++ {
		if ns.Sources[i] != ns.Sources[i-1] {
			res.SourceSwitches = append(res.SourceSwitches, SourceSwitch{
				NodeKey:     ns.NodeKey,
				FromSource:  ns.Sources[i-1],
				ToSource:    ns.Sources[i],
				AtSequence:  ns.Sequences[i],
				OffsetDelta: ns.Offsets[i] - ns.Offsets[i-1],
			})
		}
	}
}

// earliestAnomalous 返回异常节点中拓扑层级最上游者（无上游祖先异常者）。
func (m *Manager) earliestAnomalous(links []*model.Link, anomalous []string) string {
	if len(anomalous) == 0 {
		return ""
	}
	if len(anomalous) == 1 {
		return anomalous[0]
	}
	g := topology.BuildGraph(links)
	anomSet := map[string]bool{}
	for _, n := range anomalous {
		anomSet[n] = true
	}
	// 若某异常节点的上游也是异常节点，则该节点不是最早；反之保留。
	var candidates []string
	for _, n := range anomalous {
		hasAnomalousUpstream := false
		for _, u := range g.Upstream(n) {
			if anomSet[u] {
				hasAnomalousUpstream = true
				break
			}
		}
		if !hasAnomalousUpstream {
			candidates = append(candidates, n)
		}
	}
	if len(candidates) > 0 {
		sort.Strings(candidates)
		return candidates[0]
	}
	sort.Strings(anomalous)
	return anomalous[0]
}

func groupSamples(samples []*model.Sample) map[string][]*model.Sample {
	out := map[string][]*model.Sample{}
	for _, s := range samples {
		out[s.NodeKey] = append(out[s.NodeKey], s)
	}
	return out
}

// DefaultJumpThresholdNs 默认跳变判定阈值：1 毫秒。
const DefaultJumpThresholdNs = int64(1_000_000)
