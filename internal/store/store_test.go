package store

import (
	"testing"
	"time"

	"task210-clocktrace/internal/model"
)

func TestSnapshotRoundTrip(t *testing.T) {
	db, err := Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	snap, err := db.CreateSnapshot("roundtrip")
	if err != nil {
		t.Fatal(err)
	}
	if snap.ID == 0 {
		t.Fatal("expected non-zero id")
	}

	got, err := db.GetSnapshot(snap.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "roundtrip" || got.Status != model.SnapshotCollecting {
		t.Fatalf("unexpected snapshot: %+v", got)
	}

	list, err := db.ListSnapshots()
	if err != nil || len(list) != 1 {
		t.Fatalf("expected 1 snapshot, got %d err=%v", len(list), err)
	}
}

func TestSampleIdempotent(t *testing.T) {
	db, err := Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	snap, _ := db.CreateSnapshot("s")
	node := &model.Node{SnapshotID: snap.ID, NodeKey: "n1", Role: "ordinary", ClockClass: 255}
	if _, err := db.InsertNode(node); err != nil {
		t.Fatal(err)
	}

	smp := &model.Sample{
		SnapshotID: snap.ID, NodeKey: "n1", Sequence: 1,
		OffsetNs: 100, RoundtripNs: 10, Unit: "ns", Status: model.SampleValid,
		CollectedAt: time.Now().UTC(),
	}
	if _, err := db.InsertSample(smp); err != nil {
		t.Fatal(err)
	}
	// 重复插入应命中 UNIQUE(node_key,sequence)。
	if _, err := db.InsertSample(smp); err != model.ErrDuplicate {
		t.Fatalf("expected ErrDuplicate, got %v", err)
	}
}

func TestEventSealRoundTrip(t *testing.T) {
	db, err := Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	snap, _ := db.CreateSnapshot("s")
	ev, err := db.InsertEvent(snap.ID, "summary")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.UpdateEventStatus(ev.ID, model.EventLocalizing); err != nil {
		t.Fatal(err)
	}
	if err := db.SealEvent(ev.ID); err != nil {
		t.Fatal(err)
	}
	got, err := db.GetEvent(ev.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != model.EventSealed {
		t.Fatalf("expected sealed, got %s", got.Status)
	}
}
