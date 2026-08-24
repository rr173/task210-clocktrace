// Package topology 维护网络快照的节点与同步层级边，并提供环检测。
package topology

import (
	"fmt"

	"task210-clocktrace/internal/model"
	"task210-clocktrace/internal/store"
)

// Manager 拓扑管理器。
type Manager struct {
	db *store.DB
}

// New 创建拓扑管理器。
func New(db *store.DB) *Manager { return &Manager{db: db} }

// CreateSnapshot 创建网络快照（状态 collecting）。
func (m *Manager) CreateSnapshot(name string) (*model.Snapshot, error) {
	if name == "" {
		return nil, fmt.Errorf("snapshot name required")
	}
	return m.db.CreateSnapshot(name)
}

// GetSnapshot 查询快照。
func (m *Manager) GetSnapshot(id int64) (*model.Snapshot, error) { return m.db.GetSnapshot(id) }

// ListSnapshots 列出快照。
func (m *Manager) ListSnapshots() ([]*model.Snapshot, error) { return m.db.ListSnapshots() }

// LockSnapshot 将快照从 collecting 锁定为 analyzable。
func (m *Manager) LockSnapshot(id int64) (*model.Snapshot, error) {
	s, err := m.db.GetSnapshot(id)
	if err != nil {
		return nil, err
	}
	if s.Status != model.SnapshotCollecting {
		return nil, fmt.Errorf("%w: snapshot is %s", model.ErrInvalidState, s.Status)
	}
	if err := m.db.UpdateSnapshotStatus(id, model.SnapshotAnalyzable); err != nil {
		return nil, err
	}
	return m.db.GetSnapshot(id)
}

// ArchiveSnapshot 归档快照。
func (m *Manager) ArchiveSnapshot(id int64) (*model.Snapshot, error) {
	s, err := m.db.GetSnapshot(id)
	if err != nil {
		return nil, err
	}
	if s.Status == model.SnapshotArchived {
		return s, nil
	}
	if err := m.db.ArchiveSnapshot(id); err != nil {
		return nil, err
	}
	return m.db.GetSnapshot(id)
}

// AddNode 在快照下登记节点（仅 collecting 状态可写）。
func (m *Manager) AddNode(snapshotID int64, nodeKey, role, hostname string, clockClass int) (*model.Node, error) {
	if err := m.requireWritable(snapshotID); err != nil {
		return nil, err
	}
	if nodeKey == "" {
		return nil, fmt.Errorf("node key required")
	}
	if role == "" {
		role = "ordinary"
	}
	n := &model.Node{
		SnapshotID: snapshotID,
		NodeKey:    nodeKey,
		Role:       role,
		Hostname:   hostname,
		ClockClass: clockClass,
	}
	return m.db.InsertNode(n)
}

// AddLink 在快照下登记同步层级边；若引入未声明的环则拒绝。
func (m *Manager) AddLink(snapshotID int64, from, to, protocol string, declared bool) (*model.Link, error) {
	if err := m.requireWritable(snapshotID); err != nil {
		return nil, err
	}
	if from == "" || to == "" {
		return nil, fmt.Errorf("both from_node and to_node required")
	}
	if from == to {
		return nil, model.ErrTopologyCycle
	}
	// 校验两端节点存在。
	if ok, err := m.db.NodeExists(snapshotID, from); err != nil || !ok {
		return nil, model.ErrUnknownNode
	}
	if ok, err := m.db.NodeExists(snapshotID, to); err != nil || !ok {
		return nil, model.ErrUnknownNode
	}
	// 环检测：加入 from->to 后，若 to 可到达 from 则成环。
	links, err := m.db.ListLinks(snapshotID)
	if err != nil {
		return nil, err
	}
	g := BuildGraph(links)
	if g.Reachable(to, from) {
		if !declared {
			return nil, model.ErrTopologyCycle
		}
	}
	l := &model.Link{
		SnapshotID: snapshotID,
		FromNode:   from,
		ToNode:     to,
		Protocol:   protocol,
		Declared:   declared,
	}
	if protocol == "" {
		l.Protocol = "ntp"
	}
	return m.db.InsertLink(l)
}

// ValidateSnapshot 校验快照拓扑：所有边两端节点存在、无未声明环。
func (m *Manager) ValidateSnapshot(snapshotID int64) error {
	links, err := m.db.ListLinks(snapshotID)
	if err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, l := range links {
		for _, n := range []string{l.FromNode, l.ToNode} {
			if !seen[n] {
				ok, err := m.db.NodeExists(snapshotID, n)
				if err != nil {
					return err
				}
				if !ok {
					return fmt.Errorf("%w: %s", model.ErrUnknownNode, n)
				}
				seen[n] = true
			}
		}
	}
	if HasCycle(links) {
		return model.ErrTopologyCycle
	}
	return nil
}

// requireWritable 确保快照处于 collecting 状态。
func (m *Manager) requireWritable(snapshotID int64) error {
	s, err := m.db.GetSnapshot(snapshotID)
	if err != nil {
		return err
	}
	switch s.Status {
	case model.SnapshotCollecting, model.SnapshotAnalyzable:
		return nil
	case model.SnapshotArchived:
		return model.ErrSnapshotArchived
	default:
		return fmt.Errorf("%w: snapshot is %s", model.ErrInvalidState, s.Status)
	}
}
