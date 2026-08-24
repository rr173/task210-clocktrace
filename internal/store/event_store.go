package store

import (
	"time"

	"task210-clocktrace/internal/model"
)

// InsertEvent 创建一个漂移事件。
func (db *DB) InsertEvent(snapshotID int64, summary string) (*model.DriftEvent, error) {
	now := nowText()
	res, err := db.conn.Exec(
		`INSERT INTO drift_events(snapshot_id, status, summary, created_at, revision)
		 VALUES(?, ?, ?, ?, 1)`,
		snapshotID, model.EventObserved, summary, now,
	)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	created, _ := time.Parse(time.RFC3339Nano, now)
	return &model.DriftEvent{ID: id, SnapshotID: snapshotID, Status: model.EventObserved, Summary: summary, CreatedAt: created, Revision: 1}, nil
}

// GetEvent 按 ID 查询漂移事件。
func (db *DB) GetEvent(id int64) (*model.DriftEvent, error) {
	row := db.conn.QueryRow(
		`SELECT id, snapshot_id, status, summary, created_at, COALESCE(sealed_at,''), revision FROM drift_events WHERE id = ?`, id,
	)
	var e model.DriftEvent
	var createdAt, sealedAt string
	if err := row.Scan(&e.ID, &e.SnapshotID, &e.Status, &e.Summary, &createdAt, &sealedAt, &e.Revision); err != nil {
		if isNoRows(err) {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	e.CreatedAt, _ = parseTime(createdAt)
	if sealedAt != "" {
		t, _ := parseTime(sealedAt)
		e.SealedAt = &t
	}
	return &e, nil
}

// ListEventsBySnapshot 列出快照下的漂移事件。
func (db *DB) ListEventsBySnapshot(snapshotID int64) ([]*model.DriftEvent, error) {
	rows, err := db.conn.Query(
		`SELECT id, snapshot_id, status, summary, created_at, COALESCE(sealed_at,''), revision
		 FROM drift_events WHERE snapshot_id = ? ORDER BY id DESC`,
		snapshotID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.DriftEvent
	for rows.Next() {
		var e model.DriftEvent
		var createdAt, sealedAt string
		if err := rows.Scan(&e.ID, &e.SnapshotID, &e.Status, &e.Summary, &createdAt, &sealedAt, &e.Revision); err != nil {
			return nil, err
		}
		e.CreatedAt, _ = parseTime(createdAt)
		if sealedAt != "" {
			t, _ := parseTime(sealedAt)
			e.SealedAt = &t
		}
		out = append(out, &e)
	}
	return out, rows.Err()
}

// ListAllEvents 列出全部漂移事件（统计用）。
func (db *DB) ListAllEvents() ([]*model.DriftEvent, error) {
	rows, err := db.conn.Query(
		`SELECT id, snapshot_id, status, summary, created_at, COALESCE(sealed_at,''), revision
		 FROM drift_events ORDER BY id DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.DriftEvent
	for rows.Next() {
		var e model.DriftEvent
		var createdAt, sealedAt string
		if err := rows.Scan(&e.ID, &e.SnapshotID, &e.Status, &e.Summary, &createdAt, &sealedAt, &e.Revision); err != nil {
			return nil, err
		}
		e.CreatedAt, _ = parseTime(createdAt)
		if sealedAt != "" {
			t, _ := parseTime(sealedAt)
			e.SealedAt = &t
		}
		out = append(out, &e)
	}
	return out, rows.Err()
}

// UpdateEventStatus 更新事件状态并递增修订号。
func (db *DB) UpdateEventStatus(id int64, status string) error {
	res, err := db.conn.Exec(
		`UPDATE drift_events SET status = ?, revision = revision + 1 WHERE id = ?`, status, id,
	)
	if err != nil {
		return err
	}
	return checkRows(res)
}

// SealEvent 封存事件（记录封存时间，只读）。
func (db *DB) SealEvent(id int64) error {
	res, err := db.conn.Exec(
		`UPDATE drift_events SET status = ?, sealed_at = ?, revision = revision + 1 WHERE id = ?`,
		model.EventSealed, nowText(), id,
	)
	if err != nil {
		return err
	}
	return checkRows(res)
}
