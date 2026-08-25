// Package verdict 管理根因候选的确认 / 否决、样本不可信标记与事件封存。
package verdict

import (
	"fmt"
	"time"

	"task210-clocktrace/internal/model"
	"task210-clocktrace/internal/store"
)

// Manager 裁决管理器。
type Manager struct {
	db *store.DB
}

// New 创建裁决管理器。
func New(db *store.DB) *Manager { return &Manager{db: db} }

// Confirm 确认一个待确认候选，并把事件推进到 confirmed。
func (m *Manager) Confirm(eventID, candidateID int64, note string) error {
	ev, err := m.db.GetEvent(eventID)
	if err != nil {
		return err
	}
	if ev.Status == model.EventSealed {
		return model.ErrSealed
	}
	if ev.Status != model.EventLocalizing && ev.Status != model.EventConfirmed {
		return fmt.Errorf("%w: event is %s", model.ErrInvalidState, ev.Status)
	}
	c, err := m.db.GetCandidate(candidateID)
	if err != nil {
		return err
	}
	if c.EventID != eventID {
		return fmt.Errorf("candidate %d does not belong to event %d", candidateID, eventID)
	}
	// 候选一旦被否决即为终态，不能再被确认；已确认的候选同样不可重复确认。
	if !model.CanTransitionCandidate(c.Status, model.CandidateConfirmed) {
		return fmt.Errorf("%w: candidate is %s", model.ErrInvalidState, c.Status)
	}
	if err := m.db.UpdateCandidateStatus(candidateID, model.CandidateConfirmed); err != nil {
		return err
	}
	if ev.Status != model.EventConfirmed {
		if err := m.db.UpdateEventStatus(eventID, model.EventConfirmed); err != nil {
			return err
		}
	}
	_, err = m.db.InsertVerdict(&model.Verdict{
		EventID:     eventID,
		CandidateID: candidateID,
		Action:      model.ActionConfirm,
		Note:        note,
		CreatedAt:   time.Now().UTC(),
	})
	return err
}

// Reject 否决一个待确认候选。
func (m *Manager) Reject(eventID, candidateID int64, note string) error {
	ev, err := m.db.GetEvent(eventID)
	if err != nil {
		return err
	}
	if ev.Status == model.EventSealed {
		return model.ErrSealed
	}
	c, err := m.db.GetCandidate(candidateID)
	if err != nil {
		return err
	}
	if c.EventID != eventID {
		return fmt.Errorf("candidate %d does not belong to event %d", candidateID, eventID)
	}
	if !model.CanTransitionCandidate(c.Status, model.CandidateRejected) {
		return fmt.Errorf("%w: candidate is %s", model.ErrInvalidState, c.Status)
	}
	if err := m.db.UpdateCandidateStatus(candidateID, model.CandidateRejected); err != nil {
		return err
	}
	_, err = m.db.InsertVerdict(&model.Verdict{
		EventID:     eventID,
		CandidateID: candidateID,
		Action:      model.ActionReject,
		Note:        note,
		CreatedAt:   time.Now().UTC(),
	})
	return err
}

// MarkUntrusted 将样本标记为测量不可信（异常），并记录裁决。
func (m *Manager) MarkUntrusted(eventID, sampleID int64, note string) error {
	ev, err := m.db.GetEvent(eventID)
	if err != nil {
		return err
	}
	if ev.Status == model.EventSealed {
		return model.ErrSealed
	}
	if _, err := m.db.GetSample(sampleID); err != nil {
		return err
	}
	if err := m.db.UpdateSampleStatus(sampleID, model.SampleAnomaly); err != nil {
		return err
	}
	_, err = m.db.InsertVerdict(&model.Verdict{
		EventID:     eventID,
		CandidateID: 0,
		Action:      model.ActionMarkUntrusted,
		Note:        note,
		CreatedAt:   time.Now().UTC(),
	})
	return err
}
