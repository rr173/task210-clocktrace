package diagnosis_test

import (
	"testing"
	"time"

	"task210-clocktrace/internal/diagnosis"
	"task210-clocktrace/internal/model"
	"task210-clocktrace/internal/service"
	"task210-clocktrace/internal/store"
)

func TestDiagnoseGeneratesCandidates(t *testing.T) {
	db, err := store.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	app := service.New(db)

	snap, _ := app.Topology.CreateSnapshot("t")
	for _, n := range []struct{ key, role string }{
		{"source", "grandmaster"}, {"edge1", "boundary"}, {"leaf", "ordinary"},
	} {
		_, _ = app.Topology.AddNode(snap.ID, n.key, n.role, n.key, 6)
	}
	_, _ = app.Topology.AddLink(snap.ID, "source", "edge1", "ntp", true)
	_, _ = app.Topology.AddLink(snap.ID, "edge1", "leaf", "ntp", true)
	_, _ = app.Topology.LockSnapshot(snap.ID)

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
		{"leaf", 1, 0, "gm-a"},
		{"leaf", 2, 5_200_000, "gm-a"},
	}
	for _, sp := range specs {
		_, _ = app.Samples.Submit(snap.ID, sp.node, sp.seq, sp.offset, 100_000, "ns", sp.src, time.Now().UTC())
	}

	ev, _ := db.InsertEvent(snap.ID, "summary")
	cands, err := diagnosis.New(db).Diagnose(ev.ID, 1_000_000)
	if err != nil {
		t.Fatal(err)
	}
	if len(cands) < 2 {
		t.Fatalf("expected >= 2 candidates, got %d", len(cands))
	}

	got, _ := db.GetEvent(ev.ID)
	if got.Status != model.EventLocalizing {
		t.Fatalf("expected event localizing, got %s", got.Status)
	}

	for _, c := range cands {
		paths, err := db.ListEvidencePaths(c.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(paths) == 0 {
			t.Fatalf("candidate %d should have evidence paths", c.ID)
		}
	}
}

func TestDiagnoseInsufficientEvidence(t *testing.T) {
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

	ev, _ := db.InsertEvent(snap.ID, "summary")
	cands, err := diagnosis.New(db).Diagnose(ev.ID, 1_000_000)
	if err != nil {
		t.Fatal(err)
	}
	if cands != nil {
		t.Fatalf("expected no candidates, got %d", len(cands))
	}
	got, _ := db.GetEvent(ev.ID)
	if got.Status != model.EventInsufficientEvidence {
		t.Fatalf("expected insufficient_evidence, got %s", got.Status)
	}
}
