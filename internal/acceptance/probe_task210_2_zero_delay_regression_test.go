package acceptance

import (
	"testing"
	"time"

	"task210-clocktrace/internal/service"
	"task210-clocktrace/internal/store"
)

func TestBug02_ZeroRoundtripSampleIsAccepted(t *testing.T) {
	db, err := store.Open(t.TempDir() + "/clocktrace.db")
	if err != nil { t.Fatal(err) }
	defer db.Close()
	app := service.New(db)
	snap, err := app.Topology.CreateSnapshot("zero-delay")
	if err != nil { t.Fatal(err) }
	if _, err := app.Topology.AddNode(snap.ID, "source", "grandmaster", "gm", 6); err != nil { t.Fatal(err) }
	if _, err := app.Samples.Submit(snap.ID, "source", 1, 0, 0, "ns", "gm-a", time.Now().UTC()); err != nil { t.Fatalf("zero roundtrip rejected: %v", err) }
}
