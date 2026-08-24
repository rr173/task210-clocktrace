package acceptance

import (
	"errors"
	"testing"

	"task210-clocktrace/internal/model"
	"task210-clocktrace/internal/service"
	"task210-clocktrace/internal/store"
)

func TestBug03_LockedSnapshotRejectsTopologyMutation(t *testing.T) {
	db, err := store.Open(t.TempDir() + "/clocktrace.db")
	if err != nil { t.Fatal(err) }
	defer db.Close()
	app := service.New(db)
	snap, err := app.Topology.CreateSnapshot("locked")
	if err != nil { t.Fatal(err) }
	if _, err := app.Topology.AddNode(snap.ID, "source", "grandmaster", "gm", 6); err != nil { t.Fatal(err) }
	if _, err := app.Topology.LockSnapshot(snap.ID); err != nil { t.Fatal(err) }
	if _, err := app.Topology.AddNode(snap.ID, "leaf", "ordinary", "leaf", 248); !errors.Is(err, model.ErrInvalidState) { t.Fatalf("expected invalid state, got %v", err) }
}

