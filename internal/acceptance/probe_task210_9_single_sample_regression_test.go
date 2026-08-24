package acceptance

import (
	"testing"
	"time"

	"task210-clocktrace/internal/service"
	"task210-clocktrace/internal/store"
)

func TestBug09_SingleSampleAnalysisDoesNotPanic(t *testing.T) {
	db, err := store.Open(t.TempDir() + "/clocktrace.db")
	if err != nil { t.Fatal(err) }
	defer db.Close()
	app := service.New(db)
	snap, err := app.Topology.CreateSnapshot("single")
	if err != nil { t.Fatal(err) }
	if _, err := app.Topology.AddNode(snap.ID, "source", "grandmaster", "gm", 6); err != nil { t.Fatal(err) }
	if _, err := app.Samples.Submit(snap.ID, "source", 1, 100, 100, "ns", "gm-a", time.Now().UTC()); err != nil { t.Fatal(err) }
	res, err := app.Propagation.Analyze(snap.ID, 1_000_000)
	if err != nil { t.Fatal(err) }
	if res.Series["source"] == nil { t.Fatal("missing source series") }
}
