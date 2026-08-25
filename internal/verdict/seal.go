package verdict

import (
	"fmt"
	"time"

	"task210-clocktrace/internal/model"
)

// Seal 封存一个已确认事件：事件转 sealed，之后不可再修改（只能产生修订）。
// 封存后对事件、候选、样本的写操作均被拒绝。
func (m *Manager) Seal(eventID int64, note string) error {
	ev, err := m.db.GetEvent(eventID)
	if err != nil {
		return err
	}
	switch ev.Status {
	case model.EventSealed:
		return model.ErrSealed
	case model.EventConfirmed:
		// 只有根因已确认的事件才能封存；定位中、证据不足等状态须拒绝并保持原状。
	default:
		return fmt.Errorf("%w: cannot seal event in %s (need confirmed)", model.ErrInvalidState, ev.Status)
	}
	if err := m.db.SealEvent(eventID); err != nil {
		return err
	}
	_, err = m.db.InsertVerdict(&model.Verdict{
		EventID:     eventID,
		CandidateID: 0,
		Action:      model.ActionSeal,
		Note:        note,
		CreatedAt:   time.Now().UTC(),
	})
	return err
}

// Sealed 判断事件是否已封存。
func (m *Manager) Sealed(eventID int64) (bool, error) {
	ev, err := m.db.GetEvent(eventID)
	if err != nil {
		return false, err
	}
	return ev.Status == model.EventSealed, nil
}
