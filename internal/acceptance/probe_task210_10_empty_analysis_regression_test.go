package acceptance

import (
	"testing"

	"task210-clocktrace/internal/service"
	"task210-clocktrace/internal/store"
)

func TestBug10_EmptyAnalysisReturnsInsufficientEvidence(t *testing.T) {
	db, err := store.Open(t.TempDir() + "/clocktrace.db")
	if err != nil { t.Fatal(err) }
	defer db.Close()
	app := service.New(db)
	snap, err := app.Topology.CreateSnapshot("empty")
	if err != nil { t.Fatal(err) }
	if _, err := app.Topology.AddNode(snap.ID, "source", "grandmaster", "gm", 6); err != nil { t.Fatal(err) }
	res, err := app.Analyze(snap.ID, 1_000_000)
	if err != nil { t.Fatal(err) }
	if res.Event.Status != "insufficient_evidence" || len(res.Candidates) != 0 { t.Fatalf("unexpected empty analysis: event=%+v candidates=%d", res.Event, len(res.Candidates)) }
}

