package store

import (
	"task210-clocktrace/internal/model"
)

// InsertCandidate 插入根因候选。
func (db *DB) InsertCandidate(c *model.RootCauseCandidate) (*model.RootCauseCandidate, error) {
	res, err := db.conn.Exec(
		`INSERT INTO root_cause_candidates(event_id, kind, node_key, link_id, score, status, evidence)
		 VALUES(?, ?, ?, ?, ?, ?, ?)`,
		c.EventID, c.Kind, c.NodeKey, c.LinkID, c.Score, c.Status, c.Evidence,
	)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	c.ID = id
	return c, nil
}

// GetCandidate 按 ID 查询候选。
func (db *DB) GetCandidate(id int64) (*model.RootCauseCandidate, error) {
	row := db.conn.QueryRow(
		`SELECT id, event_id, kind, node_key, link_id, score, status, evidence
		 FROM root_cause_candidates WHERE id = ?`, id,
	)
	var c model.RootCauseCandidate
	if err := row.Scan(&c.ID, &c.EventID, &c.Kind, &c.NodeKey, &c.LinkID, &c.Score, &c.Status, &c.Evidence); err != nil {
		if isNoRows(err) {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	return &c, nil
}

// ListCandidatesByEvent 列出事件下按分数降序的候选。
func (db *DB) ListCandidatesByEvent(eventID int64) ([]*model.RootCauseCandidate, error) {
	rows, err := db.conn.Query(
		`SELECT id, event_id, kind, node_key, link_id, score, status, evidence
		 FROM root_cause_candidates WHERE event_id = ? ORDER BY score DESC, id`,
		eventID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.RootCauseCandidate
	for rows.Next() {
		var c model.RootCauseCandidate
		if err := rows.Scan(&c.ID, &c.EventID, &c.Kind, &c.NodeKey, &c.LinkID, &c.Score, &c.Status, &c.Evidence); err != nil {
			return nil, err
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

// UpdateCandidateStatus 更新候选状态。
func (db *DB) UpdateCandidateStatus(id int64, status string) error {
	res, err := db.conn.Exec(`UPDATE root_cause_candidates SET status = ? WHERE id = ?`, status, id)
	if err != nil {
		return err
	}
	return checkRows(res)
}

// DeleteCandidatesByEvent 删除事件下的全部候选及其证据路径（重跑诊断前清理）。
func (db *DB) DeleteCandidatesByEvent(eventID int64) error {
	ids, err := db.candidateIDs(eventID)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if _, err := db.conn.Exec(`DELETE FROM evidence_paths WHERE candidate_id = ?`, id); err != nil {
			return err
		}
	}
	if _, err := db.conn.Exec(`DELETE FROM root_cause_candidates WHERE event_id = ?`, eventID); err != nil {
		return err
	}
	return nil
}

func (db *DB) candidateIDs(eventID int64) ([]int64, error) {
	rows, err := db.conn.Query(`SELECT id FROM root_cause_candidates WHERE event_id = ?`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// InsertEvidencePath 插入证据路径节点。
func (db *DB) InsertEvidencePath(p *model.EvidencePath) (*model.EvidencePath, error) {
	res, err := db.conn.Exec(
		`INSERT INTO evidence_paths(candidate_id, order_idx, node_key, offset_ns)
		 VALUES(?, ?, ?, ?)`,
		p.CandidateID, p.OrderIdx, p.NodeKey, p.OffsetNs,
	)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	p.ID = id
	return p, nil
}

// ListEvidencePaths 按候选列出证据路径（按顺序）。
func (db *DB) ListEvidencePaths(candidateID int64) ([]*model.EvidencePath, error) {
	rows, err := db.conn.Query(
		`SELECT id, candidate_id, order_idx, node_key, offset_ns
		 FROM evidence_paths WHERE candidate_id = ? ORDER BY order_idx`,
		candidateID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.EvidencePath
	for rows.Next() {
		var p model.EvidencePath
		if err := rows.Scan(&p.ID, &p.CandidateID, &p.OrderIdx, &p.NodeKey, &p.OffsetNs); err != nil {
			return nil, err
		}
		out = append(out, &p)
	}
	return out, rows.Err()
}

// InsertVerdict 记录一次裁决。
func (db *DB) InsertVerdict(v *model.Verdict) (*model.Verdict, error) {
	res, err := db.conn.Exec(
		`INSERT INTO verdicts(event_id, candidate_id, action, note, created_at)
		 VALUES(?, ?, ?, ?, ?)`,
		v.EventID, v.CandidateID, v.Action, v.Note, formatTime(v.CreatedAt),
	)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	v.ID = id
	return v, nil
}

// ListVerdictsByEvent 列出事件下的全部裁决。
func (db *DB) ListVerdictsByEvent(eventID int64) ([]*model.Verdict, error) {
	rows, err := db.conn.Query(
		`SELECT id, event_id, candidate_id, action, note, created_at
		 FROM verdicts WHERE event_id = ? ORDER BY id`,
		eventID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Verdict
	for rows.Next() {
		var v model.Verdict
		var createdAt string
		if err := rows.Scan(&v.ID, &v.EventID, &v.CandidateID, &v.Action, &v.Note, &createdAt); err != nil {
			return nil, err
		}
		v.CreatedAt, _ = parseTime(createdAt)
		out = append(out, &v)
	}
	return out, rows.Err()
}
