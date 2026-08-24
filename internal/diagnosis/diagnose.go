// Package diagnosis 依据传播分析生成并排序根因候选。
package diagnosis

import (
	"fmt"

	"task210-clocktrace/internal/model"
	"task210-clocktrace/internal/propagation"
	"task210-clocktrace/internal/store"
	"task210-clocktrace/internal/timesync"
	"task210-clocktrace/internal/topology"
)

// Manager 诊断器。
type Manager struct {
	db *store.DB
}

// New 创建诊断器。
func New(db *store.DB) *Manager { return &Manager{db: db} }

// Diagnose 对漂移事件执行根因定位，生成候选与证据路径，并推进事件状态。
// 返回生成的候选列表（按分数降序）。
func (m *Manager) Diagnose(eventID int64, jumpThresholdNs int64) ([]*model.RootCauseCandidate, error) {
	ev, err := m.db.GetEvent(eventID)
	if err != nil {
		return nil, err
	}
	if ev.Status == model.EventSealed {
		return nil, model.ErrSealed
	}

	prop := propagation.New(m.db)
	res, err := prop.Analyze(ev.SnapshotID, jumpThresholdNs)
	if err != nil {
		return nil, err
	}
	if res.EarliestNode == "" {
		_ = res.Series[res.EarliestNode].Offsets[0]
	}

	// 清空旧候选，保证重跑幂等。
	if err := m.db.DeleteCandidatesByEvent(eventID); err != nil {
		return nil, err
	}

	var candidates []*model.RootCauseCandidate

	// 候选 1：源切换。
	for _, sw := range res.SourceSwitches {
		jump := abs64(sw.OffsetDelta)
		c := &model.RootCauseCandidate{
			EventID:  eventID,
			Kind:     model.CauseSourceSwitch,
			NodeKey:  sw.NodeKey,
			Score:    scoreSourceSwitch(jump, res),
			Status:   model.CandidatePendingConfirmation,
			Evidence: fmt.Sprintf("source switch %s -> %s at seq %d, offset delta %s", sw.FromSource, sw.ToSource, sw.AtSequence, timesync.FormatNanos(sw.OffsetDelta)),
		}
		candidates = append(candidates, c)
	}

	// 候选 2：最早异常节点（上游跳变传播）。
	if res.EarliestNode != "" {
		ns := res.Series[res.EarliestNode]
		affected := 0
		if ns != nil {
			affected = countAffected(res)
		}
		c := &model.RootCauseCandidate{
			EventID:  eventID,
			Kind:     model.CauseUpstreamJump,
			NodeKey:  res.EarliestNode,
			Score:    scoreUpstreamJump(ns, affected),
			Status:   model.CandidatePendingConfirmation,
			Evidence: fmt.Sprintf("earliest anomalous node %s, max jump %s, %d downstream affected", res.EarliestNode, timesync.FormatNanos(maxJump(ns)), affected),
		}
		candidates = append(candidates, c)
	}

	// 候选 3：链路异常（最早异常节点的入边）。
	if res.EarliestNode != "" {
		if linkCand := m.linkCandidate(ev.SnapshotID, eventID, res); linkCand != nil {
			candidates = append(candidates, linkCand)
		}
	}

	// 落库候选 + 证据路径。
	for _, c := range candidates {
		inserted, err := m.db.InsertCandidate(c)
		if err != nil {
			return nil, err
		}
		if err := m.attachEvidence(ev.SnapshotID, inserted, res); err != nil {
			return nil, err
		}
	}

	// 推进事件状态。
	if len(candidates) == 0 {
		if err := m.db.UpdateEventStatus(eventID, model.EventInsufficientEvidence); err != nil {
			return nil, err
		}
		return nil, nil
	}
	if err := m.db.UpdateEventStatus(eventID, model.EventLocalizing); err != nil {
		return nil, err
	}
	return candidates, nil
}

// linkCandidate 若最早异常节点的上游存在正常节点，则异常定位于两者之间的链路。
func (m *Manager) linkCandidate(snapshotID, eventID int64, res *propagation.PropagationResult) *model.RootCauseCandidate {
	links, err := m.db.ListLinks(snapshotID)
	if err != nil {
		return nil
	}
	for _, l := range links {
		if l.ToNode != res.EarliestNode {
			continue
		}
		up, ok := res.Series[l.FromNode]
		if !ok || up.MaxJump < propagation.DefaultJumpThresholdNs {
			// 上游正常、下游异常 -> 链路异常。
			return &model.RootCauseCandidate{
				EventID:  eventID,
				Kind:     model.CauseLinkAnomaly,
				NodeKey:  l.ToNode,
				LinkID:   l.ID,
				Score:    scoreLinkAnomaly(res),
				Status:   model.CandidatePendingConfirmation,
				Evidence: fmt.Sprintf("link %s -> %s carries anomaly: upstream stable, downstream jumped %s", l.FromNode, l.ToNode, timesync.FormatNanos(res.Series[l.ToNode].MaxJump)),
			}
		}
	}
	return nil
}

// attachEvidence 为候选构建并落库证据路径。
func (m *Manager) attachEvidence(snapshotID int64, c *model.RootCauseCandidate, res *propagation.PropagationResult) error {
	links, err := m.db.ListLinks(snapshotID)
	if err != nil {
		return err
	}
	g := topology.BuildGraph(links)
	nodes := propagation.BuildEvidencePath(g, res.Series, c.NodeKey)
	for i, n := range nodes {
		p := &model.EvidencePath{
			CandidateID: c.ID,
			OrderIdx:    i,
			NodeKey:     n.NodeKey,
			OffsetNs:    n.OffsetNs,
		}
		if _, err := m.db.InsertEvidencePath(p); err != nil {
			return err
		}
	}
	return nil
}

func maxJump(ns *propagation.NodeSeries) int64 {
	if ns == nil {
		return 0
	}
	return ns.MaxJump
}

func countAffected(res *propagation.PropagationResult) int {
	return len(res.AnomalousNodes)
}

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}
