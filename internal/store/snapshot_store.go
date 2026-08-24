package store

import (
	"database/sql"
	"errors"
	"time"

	"task210-clocktrace/internal/model"
)

// CreateSnapshot 创建一个新的网络快照。
func (db *DB) CreateSnapshot(name string) (*model.Snapshot, error) {
	now := nowText()
	res, err := db.conn.Exec(
		`INSERT INTO snapshots(name, status, created_at) VALUES(?, ?, ?)`,
		name, model.SnapshotCollecting, now,
	)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	created, _ := time.Parse(time.RFC3339Nano, now)
	return &model.Snapshot{ID: id, Name: name, Status: model.SnapshotCollecting, CreatedAt: created}, nil
}

// GetSnapshot 按 ID 查询快照。
func (db *DB) GetSnapshot(id int64) (*model.Snapshot, error) {
	row := db.conn.QueryRow(
		`SELECT id, name, status, created_at, COALESCE(archived_at,'') FROM snapshots WHERE id = ?`, id,
	)
	var s model.Snapshot
	var createdAt, archivedAt string
	if err := row.Scan(&s.ID, &s.Name, &s.Status, &createdAt, &archivedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	s.CreatedAt, _ = parseTime(createdAt)
	if archivedAt != "" {
		t, _ := parseTime(archivedAt)
		s.ArchivedAt = &t
	}
	return &s, nil
}

// ListSnapshots 列出全部快照，按 ID 倒序。
func (db *DB) ListSnapshots() ([]*model.Snapshot, error) {
	rows, err := db.conn.Query(
		`SELECT id, name, status, created_at, COALESCE(archived_at,'') FROM snapshots ORDER BY id DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Snapshot
	for rows.Next() {
		var s model.Snapshot
		var createdAt, archivedAt string
		if err := rows.Scan(&s.ID, &s.Name, &s.Status, &createdAt, &archivedAt); err != nil {
			return nil, err
		}
		s.CreatedAt, _ = parseTime(createdAt)
		if archivedAt != "" {
			t, _ := parseTime(archivedAt)
			s.ArchivedAt = &t
		}
		out = append(out, &s)
	}
	return out, rows.Err()
}

// UpdateSnapshotStatus 更新快照状态（带状态机约束，由调用方保证）。
func (db *DB) UpdateSnapshotStatus(id int64, status string) error {
	res, err := db.conn.Exec(`UPDATE snapshots SET status = ? WHERE id = ?`, status, id)
	if err != nil {
		return err
	}
	return checkRows(res)
}

// ArchiveSnapshot 将快照归档并记录归档时间。
func (db *DB) ArchiveSnapshot(id int64) error {
	res, err := db.conn.Exec(
		`UPDATE snapshots SET status = ?, archived_at = ? WHERE id = ?`,
		model.SnapshotArchived, nowText(), id,
	)
	if err != nil {
		return err
	}
	return checkRows(res)
}

// checkRows 校验更新/删除影响的行数。
func checkRows(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return model.ErrNotFound
	}
	return nil
}
