package acceptance

import (
	"testing"
	"time"

	"task210-clocktrace/internal/service"
	"task210-clocktrace/internal/store"
)

func TestBug01_UnitNamesRemainCaseInsensitive(t *testing.T) {
	db, err := store.Open(t.TempDir() + "/clocktrace.db")
	if err != nil { t.Fatal(err) }
	defer db.Close()
	app := service.New(db)
	snap, err := app.Topology.CreateSnapshot("units")
	if err != nil { t.Fatal(err) }
	if _, err := app.Topology.AddNode(snap.ID, "source", "grandmaster", "gm", 6); err != nil { t.Fatal(err) }
	s, err := app.Samples.Submit(snap.ID, "source", 1, 2, 1, "MS", "gm-a", time.Now().UTC())
	if err != nil { t.Fatal(err) }
	if s.OffsetNs != 2_000_000 || s.RoundtripNs != 1_000_000 { t.Fatalf("unexpected conversion: %+v", s) }
}
