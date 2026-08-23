package store

import (
	"task210-clocktrace/internal/model"
)

// InsertSample 插入节点样本；同 node_key 同 sequence 已存在时返回 ErrDuplicate（幂等）。
func (db *DB) InsertSample(s *model.Sample) (*model.Sample, error) {
	res, err := db.conn.Exec(
		`INSERT INTO samples(snapshot_id, node_key, sequence, offset_ns, roundtrip_ns, unit, source_id, status, collected_at)
		 VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.SnapshotID, s.NodeKey, s.Sequence, s.OffsetNs, s.RoundtripNs, s.Unit, s.SourceID, s.Status, formatTime(s.CollectedAt),
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
	s.ID = id
	return s, nil
}

// GetSample 按 ID 查询样本。
func (db *DB) GetSample(id int64) (*model.Sample, error) {
	row := db.conn.QueryRow(
		`SELECT id, snapshot_id, node_key, sequence, offset_ns, roundtrip_ns, unit, source_id, status, collected_at
		 FROM samples WHERE id = ?`, id,
	)
	return scanSample(row)
}

// ListSamplesByNode 列出某节点按序号升序的样本（用于偏移序列分析）。
func (db *DB) ListSamplesByNode(snapshotID int64, nodeKey string) ([]*model.Sample, error) {
	rows, err := db.conn.Query(
		`SELECT id, snapshot_id, node_key, sequence, offset_ns, roundtrip_ns, unit, source_id, status, collected_at
		 FROM samples WHERE snapshot_id = ? AND node_key = ? ORDER BY sequence`,
		snapshotID, nodeKey,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Sample
	for rows.Next() {
		s, err := scanSampleRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ListSamplesBySnapshot 列出快照下全部样本，按节点与序号排序。
func (db *DB) ListSamplesBySnapshot(snapshotID int64) ([]*model.Sample, error) {
	rows, err := db.conn.Query(
		`SELECT id, snapshot_id, node_key, sequence, offset_ns, roundtrip_ns, unit, source_id, status, collected_at
		 FROM samples WHERE snapshot_id = ? ORDER BY node_key, sequence`,
		snapshotID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Sample
	for rows.Next() {
		s, err := scanSampleRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// GetSampleByNodeSequence 按节点与序号查询样本（幂等回查）。
func (db *DB) GetSampleByNodeSequence(nodeKey string, sequence int64) (*model.Sample, error) {
	row := db.conn.QueryRow(
		`SELECT id, snapshot_id, node_key, sequence, offset_ns, roundtrip_ns, unit, source_id, status, collected_at
		 FROM samples WHERE node_key = ? AND sequence = ? ORDER BY id LIMIT 1`,
		nodeKey, sequence,
	)
	return scanSample(row)
}

// UpdateSampleStatus 更新样本状态（如标记测量不可信）。
func (db *DB) UpdateSampleStatus(id int64, status string) error {
	res, err := db.conn.Exec(`UPDATE samples SET status = ? WHERE id = ?`, status, id)
	if err != nil {
		return err
	}
	return checkRows(res)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanSample(row rowScanner) (*model.Sample, error) {
	var s model.Sample
	var collectedAt string
	if err := row.Scan(&s.ID, &s.SnapshotID, &s.NodeKey, &s.Sequence, &s.OffsetNs, &s.RoundtripNs, &s.Unit, &s.SourceID, &s.Status, &collectedAt); err != nil {
		if isNoRows(err) {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	s.CollectedAt, _ = parseTime(collectedAt)
	return &s, nil
}

func scanSampleRows(rows interface{ Scan(dest ...any) error }) (*model.Sample, error) {
	return scanSample(rows)
}
