package propagation

import (
	"task210-clocktrace/internal/topology"
)

// EvidenceNode 证据路径上的一个受影响节点。
type EvidenceNode struct {
	NodeKey    string `json:"node_key"`
	OffsetNs   int64  `json:"offset_ns"`   // 跳变后的偏移（纳秒）
	Depth      int    `json:"depth"`       // 相对根因节点的层级深度
	MaxJumpNs  int64  `json:"max_jump_ns"` // 该节点自身的最大跳变
}

// BuildEvidencePath 从根因节点出发，沿拓扑下游构建受影响节点证据链（BFS）。
func BuildEvidencePath(g *topology.Graph, series map[string]*NodeSeries, earliest string) []EvidenceNode {
	if earliest == "" {
		return nil
	}
	down := g.Downstream(earliest)
	out := make([]EvidenceNode, 0, len(down)+1)

	root := EvidenceNode{NodeKey: earliest, Depth: 0}
	if ns, ok := series[earliest]; ok {
		root.OffsetNs = lastOffset(ns)
		root.MaxJumpNs = ns.MaxJump
	}
	out = append(out, root)

	depth := map[string]int{earliest: 0}
	for _, n := range down {
		d := depth[n] // 默认 0，由 BFS 顺序保证递增
		en := EvidenceNode{NodeKey: n, Depth: d}
		if ns, ok := series[n]; ok {
			en.OffsetNs = lastOffset(ns)
			en.MaxJumpNs = ns.MaxJump
		}
		out = append(out, en)
		// 传播深度给其子节点（粗略：层级=发现顺序）。
		for _, child := range g.Downstream(n) {
			if _, ok := depth[child]; !ok {
				depth[child] = d + 1
			}
		}
	}
	return out
}

func lastOffset(ns *NodeSeries) int64 {
	if len(ns.Offsets) == 0 {
		return 0
	}
	return ns.Offsets[len(ns.Offsets)-1]
}

// AffectedCount 统计证据链上实际受影响（跳变超阈值）的节点数。
func AffectedCount(nodes []EvidenceNode, threshold int64) int {
	cnt := 0
	for _, n := range nodes {
		if n.MaxJumpNs >= threshold {
			cnt++
		}
	}
	return cnt
}
