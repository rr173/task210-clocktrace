// Package store 提供基于 SQLite 的持久化层（modernc.org/sqlite，纯 Go 无 CGO）。
package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// DB 封装数据库连接与建表迁移。
type DB struct {
	conn *sql.DB
	path string
}

// Open 打开（必要时创建）SQLite 数据库并执行建表迁移。
func Open(path string) (*DB, error) {
	if path == "" {
		path = filepath.Join(".", "clocktrace.db")
	}
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create db dir: %w", err)
		}
	}
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	conn.SetMaxOpenConns(1) // 单写者，规避 SQLite 写锁竞争
	db := &DB{conn: conn, path: path}
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, err
	}
	return db, nil
}

// Close 关闭数据库连接。
func (db *DB) Close() error { return db.conn.Close() }

// Path 返回数据库文件路径。
func (db *DB) Path() string { return db.path }

// Conn 暴露底层连接，供需要事务或裸查询的场景使用。
func (db *DB) Conn() *sql.DB { return db.conn }

// migrate 执行全部建表语句（幂等）。
func (db *DB) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS snapshots (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			status TEXT NOT NULL,
			created_at TEXT NOT NULL,
			archived_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS nodes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			snapshot_id INTEGER NOT NULL,
			node_key TEXT NOT NULL,
			role TEXT NOT NULL,
			hostname TEXT NOT NULL DEFAULT '',
			clock_class INTEGER NOT NULL DEFAULT 255,
			UNIQUE(snapshot_id, node_key),
			FOREIGN KEY (snapshot_id) REFERENCES snapshots(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_nodes_snapshot ON nodes(snapshot_id)`,
		`CREATE TABLE IF NOT EXISTS links (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			snapshot_id INTEGER NOT NULL,
			from_node TEXT NOT NULL,
			to_node TEXT NOT NULL,
			protocol TEXT NOT NULL,
			declared INTEGER NOT NULL DEFAULT 1,
			UNIQUE(snapshot_id, from_node, to_node),
			FOREIGN KEY (snapshot_id) REFERENCES snapshots(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_links_snapshot ON links(snapshot_id)`,
		`CREATE TABLE IF NOT EXISTS samples (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			snapshot_id INTEGER NOT NULL,
			node_key TEXT NOT NULL,
			sequence INTEGER NOT NULL,
			offset_ns INTEGER NOT NULL,
			roundtrip_ns INTEGER NOT NULL,
			unit TEXT NOT NULL DEFAULT 'ns',
			source_id TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL,
			collected_at TEXT NOT NULL,
			UNIQUE(node_key, sequence),
			FOREIGN KEY (snapshot_id) REFERENCES snapshots(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_samples_snapshot ON samples(snapshot_id)`,
		`CREATE INDEX IF NOT EXISTS idx_samples_node ON samples(node_key)`,
		`CREATE TABLE IF NOT EXISTS drift_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			snapshot_id INTEGER NOT NULL,
			status TEXT NOT NULL,
			summary TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			sealed_at TEXT,
			revision INTEGER NOT NULL DEFAULT 1,
			FOREIGN KEY (snapshot_id) REFERENCES snapshots(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_events_snapshot ON drift_events(snapshot_id)`,
		`CREATE TABLE IF NOT EXISTS root_cause_candidates (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_id INTEGER NOT NULL,
			kind TEXT NOT NULL,
			node_key TEXT NOT NULL,
			link_id INTEGER NOT NULL DEFAULT 0,
			score REAL NOT NULL DEFAULT 0,
			status TEXT NOT NULL,
			evidence TEXT NOT NULL DEFAULT '',
			FOREIGN KEY (event_id) REFERENCES drift_events(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_candidates_event ON root_cause_candidates(event_id)`,
		`CREATE TABLE IF NOT EXISTS evidence_paths (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			candidate_id INTEGER NOT NULL,
			order_idx INTEGER NOT NULL,
			node_key TEXT NOT NULL,
			offset_ns INTEGER NOT NULL,
			FOREIGN KEY (candidate_id) REFERENCES root_cause_candidates(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_evidence_candidate ON evidence_paths(candidate_id)`,
		`CREATE TABLE IF NOT EXISTS verdicts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_id INTEGER NOT NULL,
			candidate_id INTEGER NOT NULL DEFAULT 0,
			action TEXT NOT NULL,
			note TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			FOREIGN KEY (event_id) REFERENCES drift_events(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_verdicts_event ON verdicts(event_id)`,
	}
	for _, s := range stmts {
		if _, err := db.conn.Exec(s); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	return nil
}

// formatTime 将时间序列化为可存储的 RFC3339Nano 文本。
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

// parseTime 解析存储的时间文本；空串返回零值。
func parseTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339Nano, s)
}

// nowText 返回当前时间的 RFC3339Nano 文本。
func nowText() string { return time.Now().UTC().Format(time.RFC3339Nano) }
