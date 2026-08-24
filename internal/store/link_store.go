package store

import (
	"task210-clocktrace/internal/model"
)

// InsertLink 插入拓扑边；同快照同 (from,to) 已存在时返回 ErrDuplicate。
func (db *DB) InsertLink(l *model.Link) (*model.Link, error) {
	res, err := db.conn.Exec(
		`INSERT INTO links(snapshot_id, from_node, to_node, protocol, declared)
		 VALUES(?, ?, ?, ?, ?)`,
		l.SnapshotID, l.FromNode, l.ToNode, l.Protocol, boolToInt(l.Declared),
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, model.ErrDuplicate
		}
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	l.ID = id
	return l, nil
}

// ListLinks 列出快照下的全部拓扑边。
func (db *DB) ListLinks(snapshotID int64) ([]*model.Link, error) {
	rows, err := db.conn.Query(
		`SELECT id, snapshot_id, from_node, to_node, protocol, declared
		 FROM links WHERE snapshot_id = ? ORDER BY id`,
		snapshotID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Link
	for rows.Next() {
		var l model.Link
		var declared int
		if err := rows.Scan(&l.ID, &l.SnapshotID, &l.FromNode, &l.ToNode, &l.Protocol, &declared); err != nil {
			return nil, err
		}
		l.Declared = declared != 0
		out = append(out, &l)
	}
	return out, rows.Err()
}

// GetLink 按 ID 查询拓扑边。
func (db *DB) GetLink(id int64) (*model.Link, error) {
	row := db.conn.QueryRow(
		`SELECT id, snapshot_id, from_node, to_node, protocol, declared FROM links WHERE id = ?`, id,
	)
	var l model.Link
	var declared int
	if err := row.Scan(&l.ID, &l.SnapshotID, &l.FromNode, &l.ToNode, &l.Protocol, &declared); err != nil {
		if isNoRows(err) {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	l.Declared = declared != 0
	return &l, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
