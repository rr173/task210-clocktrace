package verdict

import (
	"testing"

	"task210-clocktrace/internal/model"
	"task210-clocktrace/internal/store"
)

func TestEventStateMachine(t *testing.T) {
	if !model.CanTransitionEvent(model.EventObserved, model.EventLocalizing) {
		t.Fatal("observed -> localizing should be allowed")
	}
	if model.CanTransitionEvent(model.EventObserved, model.EventSealed) {
		t.Fatal("observed -> sealed should be rejected")
	}
	if model.CanTransitionEvent(model.EventSealed, model.EventConfirmed) {
		t.Fatal("sealed -> confirmed should be rejected")
	}
}

func TestCandidateStateMachine(t *testing.T) {
	if !model.CanTransitionCandidate(model.CandidatePendingConfirmation, model.CandidateConfirmed) {
		t.Fatal("pending -> confirmed should be allowed")
	}
	if model.CanTransitionCandidate(model.CandidateRejected, model.CandidateConfirmed) {
		t.Fatal("rejected -> confirmed should be rejected")
	}
}

func TestConfirmRejectSealFlow(t *testing.T) {
	db, err := store.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	m := New(db)

	snap, _ := db.CreateSnapshot("s")
	ev, _ := db.InsertEvent(snap.ID, "summary")
	_ = db.UpdateEventStatus(ev.ID, model.EventLocalizing)

	c1, _ := db.InsertCandidate(&model.RootCauseCandidate{
		EventID: ev.ID, Kind: model.CauseSourceSwitch, NodeKey: "n1",
		Status: model.CandidatePendingConfirmation, Score: 10,
	})
	c2, _ := db.InsertCandidate(&model.RootCauseCandidate{
		EventID: ev.ID, Kind: model.CauseUpstreamJump, NodeKey: "n2",
		Status: model.CandidatePendingConfirmation, Score: 5,
	})

	if err := m.Reject(ev.ID, c1.ID, "no path"); err != nil {
		t.Fatal(err)
	}
	if err := m.Confirm(ev.ID, c2.ID, "confirmed"); err != nil {
		t.Fatal(err)
	}
	if err := m.Seal(ev.ID, "archive"); err != nil {
		t.Fatal(err)
	}
	// 封存后再次确认应失败。
	if err := m.Confirm(ev.ID, c2.ID, "again"); err == nil {
		t.Fatal("expected confirm after seal to fail")
	}
}
