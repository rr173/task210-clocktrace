package topology

import (
	"testing"

	"task210-clocktrace/internal/model"
)

func links(edges ...[2]string) []*model.Link {
	var out []*model.Link
	for i, e := range edges {
		out = append(out, &model.Link{ID: int64(i + 1), FromNode: e[0], ToNode: e[1]})
	}
	return out
}

func TestBuildGraphReachable(t *testing.T) {
	g := BuildGraph(links([2]string{"a", "b"}, [2]string{"b", "c"}))
	if !g.Reachable("a", "c") {
		t.Fatal("a should reach c")
	}
	if g.Reachable("c", "a") {
		t.Fatal("c should not reach a")
	}
}

func TestDownstream(t *testing.T) {
	g := BuildGraph(links([2]string{"a", "b"}, [2]string{"a", "c"}, [2]string{"b", "d"}))
	ds := g.Downstream("a")
	if len(ds) != 3 {
		t.Fatalf("expected 3 downstream nodes, got %d: %v", len(ds), ds)
	}
}

func TestUpstream(t *testing.T) {
	g := BuildGraph(links([2]string{"a", "c"}, [2]string{"b", "c"}))
	up := g.Upstream("c")
	if len(up) != 2 {
		t.Fatalf("expected 2 upstream nodes, got %d: %v", len(up), up)
	}
}

func TestHasCycle(t *testing.T) {
	acyclic := links([2]string{"a", "b"}, [2]string{"b", "c"})
	if HasCycle(acyclic) {
		t.Fatal("expected no cycle")
	}
	cyclic := links([2]string{"a", "b"}, [2]string{"b", "c"}, [2]string{"c", "a"})
	if !HasCycle(cyclic) {
		t.Fatal("expected cycle")
	}
}

func TestParentCount(t *testing.T) {
	g := BuildGraph(links([2]string{"a", "c"}, [2]string{"b", "c"}))
	pc := g.ParentCount()
	if pc["c"] != 2 {
		t.Fatalf("expected c to have 2 parents, got %d", pc["c"])
	}
	if pc["a"] != 0 {
		t.Fatalf("expected a to have 0 parents, got %d", pc["a"])
	}
}
