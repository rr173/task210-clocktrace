package store

import (
	"database/sql"
	"errors"

	"task210-clocktrace/internal/model"
)

// InsertNode 插入拓扑节点；同快照同 node_key 已存在时返回 ErrDuplicate。
func (db *DB) InsertNode(n *model.Node) (*model.Node, error) {
	res, err := db.conn.Exec(
		`INSERT INTO nodes(snapshot_id, node_key, role, hostname, clock_class)
		 VALUES(?, ?, ?, ?, ?)`,
		n.SnapshotID, n.NodeKey, n.Role, n.Hostname, n.ClockClass,
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
	n.ID = id
	return n, nil
}

// GetNode 按快照与 node_key 查询节点。
func (db *DB) GetNode(snapshotID int64, nodeKey string) (*model.Node, error) {
	row := db.conn.QueryRow(
		`SELECT id, snapshot_id, node_key, role, hostname, clock_class
		 FROM nodes WHERE snapshot_id = ? AND node_key = ?`,
		snapshotID, nodeKey,
	)
	var n model.Node
	if err := row.Scan(&n.ID, &n.SnapshotID, &n.NodeKey, &n.Role, &n.Hostname, &n.ClockClass); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	return &n, nil
}

// ListNodes 列出快照下的全部节点。
func (db *DB) ListNodes(snapshotID int64) ([]*model.Node, error) {
	rows, err := db.conn.Query(
		`SELECT id, snapshot_id, node_key, role, hostname, clock_class
		 FROM nodes WHERE snapshot_id = ? ORDER BY id`,
		snapshotID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Node
	for rows.Next() {
		var n model.Node
		if err := rows.Scan(&n.ID, &n.SnapshotID, &n.NodeKey, &n.Role, &n.Hostname, &n.ClockClass); err != nil {
			return nil, err
		}
		out = append(out, &n)
	}
	return out, rows.Err()
}

// NodeExists 判断节点是否存在。
func (db *DB) NodeExists(snapshotID int64, nodeKey string) (bool, error) {
	_, err := db.GetNode(snapshotID, nodeKey)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, model.ErrNotFound) {
		return false, nil
	}
	return false, err
}
