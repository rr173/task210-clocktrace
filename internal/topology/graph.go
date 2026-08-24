package topology

import (
	"task210-clocktrace/internal/model"
)

// Graph 有向图邻接表：键为上游节点，值为下游节点集合。
type Graph struct {
	adj map[string][]string
}

// BuildGraph 由拓扑边构建有向图。
func BuildGraph(links []*model.Link) *Graph {
	g := &Graph{adj: map[string][]string{}}
	for _, l := range links {
		g.adj[l.FromNode] = append(g.adj[l.FromNode], l.ToNode)
		if _, ok := g.adj[l.ToNode]; !ok {
			g.adj[l.ToNode] = []string{}
		}
	}
	return g
}

// Nodes 返回图全部节点。
func (g *Graph) Nodes() []string {
	out := make([]string, 0, len(g.adj))
	for n := range g.adj {
		out = append(out, n)
	}
	return out
}

// Reachable 判断从 from 出发能否到达 to（BFS）。
func (g *Graph) Reachable(from, to string) bool {
	if from == to {
		return true
	}
	visited := map[string]bool{}
	queue := []string{from}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if visited[cur] {
			continue
		}
		visited[cur] = true
		for _, next := range g.adj[cur] {
			if next == to {
				return true
			}
			if !visited[next] {
				queue = append(queue, next)
			}
		}
	}
	return false
}

// Downstream 返回 from 的全部下游节点（BFS 可达集，不含自身），按发现顺序。
func (g *Graph) Downstream(from string) []string {
	visited := map[string]bool{from: true}
	var order []string
	queue := []string{from}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, next := range g.adj[cur] {
			if visited[next] {
				continue
			}
			visited[next] = true
			order = append(order, next)
			queue = append(queue, next)
		}
	}
	return order
}

// Upstream 返回 to 的全部上游节点（逆邻接可达集，不含自身）。
func (g *Graph) Upstream(to string) []string {
	rev := map[string][]string{}
	for n, children := range g.adj {
		for _, c := range children {
			rev[c] = append(rev[c], n)
		}
	}
	visited := map[string]bool{to: true}
	var order []string
	queue := []string{to}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, prev := range rev[cur] {
			if visited[prev] {
				continue
			}
			visited[prev] = true
			order = append(order, prev)
			queue = append(queue, prev)
		}
	}
	return order
}

// HasCycle 用三色 DFS 检测有向图是否存在环。
func HasCycle(links []*model.Link) bool {
	g := BuildGraph(links)
	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := map[string]int{}
	var visit func(n string) bool
	visit = func(n string) bool {
		color[n] = gray
		for _, next := range g.adj[n] {
			switch color[next] {
			case gray:
				return true
			case white:
				if visit(next) {
					return true
				}
			}
		}
		color[n] = black
		return false
	}
	for _, n := range g.Nodes() {
		if color[n] == white {
			if visit(n) {
				return true
			}
		}
	}
	return false
}

// ParentCount 返回每个节点的上游数量（多父检测）。
func (g *Graph) ParentCount() map[string]int {
	pc := map[string]int{}
	for n := range g.adj {
		if _, ok := pc[n]; !ok {
			pc[n] = 0
		}
	}
	for _, children := range g.adj {
		for _, c := range children {
			pc[c]++
		}
	}
	return pc
}
