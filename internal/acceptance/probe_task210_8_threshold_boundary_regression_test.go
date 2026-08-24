package acceptance

import (
	"testing"
	"time"

	"task210-clocktrace/internal/service"
	"task210-clocktrace/internal/store"
)

func TestBug08_ThresholdBoundaryIsAnomalous(t *testing.T) {
	db, err := store.Open(t.TempDir() + "/clocktrace.db")
	if err != nil { t.Fatal(err) }
	defer db.Close()
	app := service.New(db)
	snap, err := app.Topology.CreateSnapshot("threshold")
	if err != nil { t.Fatal(err) }
	if _, err := app.Topology.AddNode(snap.ID, "source", "grandmaster", "gm", 6); err != nil { t.Fatal(err) }
	if _, err := app.Samples.Submit(snap.ID, "source", 1, 0, 100, "ns", "gm-a", time.Now().UTC()); err != nil { t.Fatal(err) }
	if _, err := app.Samples.Submit(snap.ID, "source", 2, 1_000_000, 100, "ns", "gm-a", time.Now().UTC()); err != nil { t.Fatal(err) }
	res, err := app.Analyze(snap.ID, 1_000_000)
	if err != nil { t.Fatal(err) }
	if res.Propagation.EarliestNode != "source" || len(res.Candidates) == 0 { t.Fatalf("threshold jump missed: %+v", res.Propagation) }
}
