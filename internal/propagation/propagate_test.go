package propagation_test

import (
	"testing"
	"time"

	"task210-clocktrace/internal/propagation"
	"task210-clocktrace/internal/service"
	"task210-clocktrace/internal/store"
)

func buildTopology(t *testing.T) (*store.DB, int64) {
	t.Helper()
	db, err := store.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	app := service.New(db)
	snap, err := app.Topology.CreateSnapshot("t")
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range []struct{ key, role string }{
		{"source", "grandmaster"}, {"edge1", "boundary"}, {"edge2", "boundary"}, {"leaf", "ordinary"},
	} {
		if _, err := app.Topology.AddNode(snap.ID, n.key, n.role, n.key, 6); err != nil {
			t.Fatal(err)
		}
	}
	for _, e := range []struct{ from, to string }{
		{"source", "edge1"}, {"edge1", "edge2"}, {"edge2", "leaf"},
	} {
		if _, err := app.Topology.AddLink(snap.ID, e.from, e.to, "ntp", true); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := app.Topology.LockSnapshot(snap.ID); err != nil {
		t.Fatal(err)
	}
	specs := []struct {
		node   string
		seq    int64
		offset int64
		src    string
	}{
		{"source", 1, 0, "gm-a"},
		{"source", 2, 5_000_000, "gm-a"},
		{"edge1", 1, 0, "gm-a"},
		{"edge1", 2, 5_100_000, "gm-b"},
		{"edge2", 1, 0, "gm-a"},
		{"edge2", 2, 5_200_000, "gm-a"},
		{"leaf", 1, 0, "gm-a"},
		{"leaf", 2, 5_300_000, "gm-a"},
	}
	for _, sp := range specs {
		if _, err := app.Samples.Submit(snap.ID, sp.node, sp.seq, sp.offset, 100_000, "ns", sp.src, time.Now().UTC()); err != nil {
			t.Fatal(err)
		}
	}
	return db, snap.ID
}

func TestAnalyzeEarliestNode(t *testing.T) {
	db, snapshotID := buildTopology(t)
	defer db.Close()

	m := propagation.New(db)
	res, err := m.Analyze(snapshotID, 1_000_000)
	if err != nil {
		t.Fatal(err)
	}
	if res.EarliestNode != "source" {
		t.Fatalf("expected earliest node 'source', got %q", res.EarliestNode)
	}
	if len(res.AnomalousNodes) != 4 {
		t.Fatalf("expected 4 anomalous nodes, got %d", len(res.AnomalousNodes))
	}
	if len(res.SourceSwitches) != 1 {
		t.Fatalf("expected 1 source switch, got %d", len(res.SourceSwitches))
	}
}

func TestAnalyzeNoAnomaly(t *testing.T) {
	db, err := store.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	app := service.New(db)
	snap, _ := app.Topology.CreateSnapshot("quiet")
	_, _ = app.Topology.AddNode(snap.ID, "a", "ordinary", "a", 6)
	_, _ = app.Topology.AddNode(snap.ID, "b", "ordinary", "b", 6)
	_, _ = app.Topology.AddLink(snap.ID, "a", "b", "ntp", true)
	_, _ = app.Topology.LockSnapshot(snap.ID)
	_, _ = app.Samples.Submit(snap.ID, "a", 1, 0, 100, "ns", "gm", time.Now().UTC())
	_, _ = app.Samples.Submit(snap.ID, "b", 1, 10, 100, "ns", "gm", time.Now().UTC())

	res, err := propagation.New(db).Analyze(snap.ID, 1_000_000)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.AnomalousNodes) != 0 {
		t.Fatalf("expected no anomaly, got %v", res.AnomalousNodes)
	}
	if res.EarliestNode != "" {
		t.Fatalf("expected empty earliest node, got %q", res.EarliestNode)
	}
}
