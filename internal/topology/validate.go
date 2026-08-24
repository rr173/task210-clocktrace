package topology

import (
	"task210-clocktrace/internal/model"
)

// TopologyReport 拓扑结构诊断报告。
type TopologyReport struct {
	Sources        []string       `json:"sources"`         // 时钟源（无上游的节点）
	IsolatedNodes  []string       `json:"isolated_nodes"`  // 无任何边连接的节点
	MultiParent    map[string]int `json:"multi_parent"`    // 存在多个上游的节点 -> 上游数
	Depths         map[string]int `json:"depths"`          // 节点到时钟源的层级深度
	EdgeCount      int            `json:"edge_count"`
	NodeCount      int            `json:"node_count"`
	HasCycle       bool           `json:"has_cycle"`
}

// Inspect 对快照拓扑做结构诊断（时钟源、孤立节点、多父、层级深度、环）。
func (m *Manager) Inspect(snapshotID int64) (*TopologyReport, error) {
	links, err := m.db.ListLinks(snapshotID)
	if err != nil {
		return nil, err
	}
	nodes, err := m.db.ListNodes(snapshotID)
	if err != nil {
		return nil, err
	}
	g := BuildGraph(links)
	parentCount := g.ParentCount()

	report := &TopologyReport{
		MultiParent: map[string]int{},
		Depths:      map[string]int{},
		EdgeCount:   len(links),
		NodeCount:   len(nodes),
		HasCycle:    HasCycle(links),
	}

	degree := map[string]int{}
	for _, l := range links {
		degree[l.FromNode]++
		degree[l.ToNode]++
	}

	for _, n := range nodes {
		if parentCount[n.NodeKey] == 0 {
			report.Sources = append(report.Sources, n.NodeKey)
		}
		if parentCount[n.NodeKey] > 1 {
			report.MultiParent[n.NodeKey] = parentCount[n.NodeKey]
		}
		if degree[n.NodeKey] == 0 {
			report.IsolatedNodes = append(report.IsolatedNodes, n.NodeKey)
		}
	}

	// BFS 计算层级深度（从时钟源出发）。
	for _, src := range report.Sources {
		m.bfsDepth(g, src, report.Depths)
	}
	return report, nil
}

func (m *Manager) bfsDepth(g *Graph, src string, depths map[string]int) {
	visited := map[string]bool{}
	queue := []string{src}
	depths[src] = 0
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if visited[cur] {
			continue
		}
		visited[cur] = true
		for _, next := range g.adj[cur] {
			if d, ok := depths[next]; !ok || d > depths[cur]+1 {
				depths[next] = depths[cur] + 1
			}
			if !visited[next] {
				queue = append(queue, next)
			}
		}
	}
}

// Links 返回快照全部拓扑边。
func (m *Manager) Links(snapshotID int64) ([]*model.Link, error) { return m.db.ListLinks(snapshotID) }

// Nodes 返回快照全部节点。
func (m *Manager) Nodes(snapshotID int64) ([]*model.Node, error) { return m.db.ListNodes(snapshotID) }

// Graph 构建快照的有向图。
func (m *Manager) Graph(snapshotID int64) (*Graph, error) {
	links, err := m.db.ListLinks(snapshotID)
	if err != nil {
		return nil, err
	}
	return BuildGraph(links), nil
}
