package acceptance

import (
	"testing"
	"time"

	"task210-clocktrace/internal/service"
	"task210-clocktrace/internal/store"
)

func TestBug07_StatsExcludeRejectedCandidatesFromConfirmedCount(t *testing.T) {
	db, err := store.Open(t.TempDir() + "/clocktrace.db")
	if err != nil { t.Fatal(err) }
	defer db.Close()
	app := service.New(db)
	snap, err := app.Topology.CreateSnapshot("stats")
	if err != nil { t.Fatal(err) }
	if _, err := app.Topology.AddNode(snap.ID, "source", "grandmaster", "gm", 6); err != nil { t.Fatal(err) }
	if _, err := app.Samples.Submit(snap.ID, "source", 1, 0, 100, "ns", "gm-a", time.Now().UTC()); err != nil { t.Fatal(err) }
	if _, err := app.Samples.Submit(snap.ID, "source", 2, 2_000_000, 100, "ns", "gm-a", time.Now().UTC()); err != nil { t.Fatal(err) }
	res, err := app.Analyze(snap.ID, 1_000_000)
	if err != nil { t.Fatal(err) }
	cands, err := app.ListCandidates(res.Event.ID)
	if err != nil { t.Fatal(err) }
	if err := app.Verdict.Reject(res.Event.ID, cands[0].ID, "not supported"); err != nil { t.Fatal(err) }
	stats, err := app.ComputeStats()
	if err != nil { t.Fatal(err) }
	if stats.Confirmed != 0 { t.Fatalf("rejected candidate counted as confirmed: %d", stats.Confirmed) }
}
