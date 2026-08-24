package acceptance

import (
	"testing"
	"time"

	"task210-clocktrace/internal/service"
	"task210-clocktrace/internal/store"
)

func TestBug04_RerunningDiagnosisReplacesCandidates(t *testing.T) {
	db, err := store.Open(t.TempDir() + "/clocktrace.db")
	if err != nil { t.Fatal(err) }
	defer db.Close()
	app := service.New(db)
	snap, err := app.Topology.CreateSnapshot("rerun")
	if err != nil { t.Fatal(err) }
	if _, err := app.Topology.AddNode(snap.ID, "source", "grandmaster", "gm", 6); err != nil { t.Fatal(err) }
	if _, err := app.Samples.Submit(snap.ID, "source", 1, 0, 100, "ns", "gm-a", time.Now().UTC()); err != nil { t.Fatal(err) }
	if _, err := app.Samples.Submit(snap.ID, "source", 2, 2_000_000, 100, "ns", "gm-a", time.Now().UTC()); err != nil { t.Fatal(err) }
	res, err := app.Analyze(snap.ID, 1_000_000)
	if err != nil { t.Fatal(err) }
	before, err := app.ListCandidates(res.Event.ID)
	if err != nil { t.Fatal(err) }
	if len(before) == 0 { t.Fatal("expected candidates") }
	if _, err := app.Diagnosis.Diagnose(res.Event.ID, 1_000_000); err != nil { t.Fatal(err) }
	after, err := app.ListCandidates(res.Event.ID)
	if err != nil { t.Fatal(err) }
	if len(after) != len(before) { t.Fatalf("rerun duplicated candidates: before=%d after=%d", len(before), len(after)) }
}
