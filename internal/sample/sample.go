// Package sample 负责节点同步样本的校验、单位换算与幂等入库。
package sample

import (
	"time"

	"task210-clocktrace/internal/model"
	"task210-clocktrace/internal/store"
	"task210-clocktrace/internal/timesync"
)

// Manager 样本管理器。
type Manager struct {
	db *store.DB
}

// New 创建样本管理器。
func New(db *store.DB) *Manager { return &Manager{db: db} }

// Submit 提交一条节点同步样本。
//
// 校验顺序：快照可接收 → 节点存在 → 单位合法并换算纳秒 → 偏移可表示、往返延迟非负。
// 重复样本（同节点同序号）幂等返回已入库样本，不报错。
func (m *Manager) Submit(snapshotID int64, nodeKey string, sequence int64, offset, roundtrip int64, unit, sourceID string, collectedAt time.Time) (*model.Sample, error) {
	snap, err := m.db.GetSnapshot(snapshotID)
	if err != nil {
		return nil, err
	}
	if snap.Status == model.SnapshotArchived {
		return nil, model.ErrSnapshotArchived
	}
	if ok, err := m.db.NodeExists(snapshotID, nodeKey); err != nil {
		return nil, err
	} else if !ok {
		return nil, model.ErrUnknownNode
	}

	u, err := timesync.ParseUnit(unit)
	if err != nil {
		return nil, err
	}
	offsetNs, err := timesync.ToNanos(offset, u)
	if err != nil {
		return nil, err
	}
	roundtripNs, err := timesync.ToNanos(roundtrip, u)
	if err != nil {
		return nil, err
	}
	if offsetNs > maxOffsetNs || offsetNs < -maxOffsetNs {
		return nil, model.ErrOffsetOverflow
	}
	if roundtripNs <= 0 {
		return nil, model.ErrNegativeDelay
	}

	s := &model.Sample{
		SnapshotID:  snapshotID,
		NodeKey:     nodeKey,
		Sequence:    sequence,
		OffsetNs:    offsetNs,
		RoundtripNs: roundtripNs,
		Unit:        string(u),
		SourceID:    sourceID,
		Status:      model.SampleValid,
		CollectedAt: collectedAt,
	}
	if collectedAt.IsZero() {
		s.CollectedAt = time.Now().UTC()
	}

	inserted, err := m.db.InsertSample(s)
	if err == nil {
		return inserted, nil
	}
	if err == model.ErrDuplicate {
		existing, gerr := m.db.GetSampleByNodeSequence(nodeKey, sequence)
		if gerr != nil {
			return nil, gerr
		}
		return existing, nil
	}
	return nil, err
}

// MarkUntrusted 将样本标记为测量不可信（异常）。
func (m *Manager) MarkUntrusted(sampleID int64) error {
	return m.db.UpdateSampleStatus(sampleID, model.SampleAnomaly)
}

// ListByNode 列出某节点的样本序列（按序号）。
func (m *Manager) ListByNode(snapshotID int64, nodeKey string) ([]*model.Sample, error) {
	return m.db.ListSamplesByNode(snapshotID, nodeKey)
}

// ListBySnapshot 列出快照全部样本。
func (m *Manager) ListBySnapshot(snapshotID int64) ([]*model.Sample, error) {
	return m.db.ListSamplesBySnapshot(snapshotID)
}

// Get 按 ID 查询样本。
func (m *Manager) Get(id int64) (*model.Sample, error) { return m.db.GetSample(id) }

// maxOffsetNs 偏移可表示上限（约 292 年，覆盖任意时钟漂移）。
const maxOffsetNs = int64(9.2e18)
