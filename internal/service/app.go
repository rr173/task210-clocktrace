// Package service 编排各业务模块，提供统一的端到端能力与统计。
package service

import (
	"fmt"

	"task210-clocktrace/internal/diagnosis"
	"task210-clocktrace/internal/model"
	"task210-clocktrace/internal/propagation"
	"task210-clocktrace/internal/sample"
	"task210-clocktrace/internal/store"
	"task210-clocktrace/internal/topology"
	"task210-clocktrace/internal/verdict"
)

// App 聚合全部业务管理器。
type App struct {
	DB          *store.DB
	Topology    *topology.Manager
	Samples     *sample.Manager
	Propagation *propagation.Manager
	Diagnosis   *diagnosis.Manager
	Verdict     *verdict.Manager
}

// New 构建应用编排层。
func New(db *store.DB) *App {
	return &App{
		DB:          db,
		Topology:    topology.New(db),
		Samples:     sample.New(db),
		Propagation: propagation.New(db),
		Diagnosis:   diagnosis.New(db),
		Verdict:     verdict.New(db),
	}
}

// AnalyzeResult 一次分析的完整结果。
type AnalyzeResult struct {
	Event      *model.DriftEvent
	Candidates []*model.RootCauseCandidate
	Propagation *propagation.PropagationResult
}

// Analyze 对快照执行完整根因定位闭环：
// 锁定快照（若仍在收集）→ 创建漂移事件 → 运行诊断生成候选。
func (a *App) Analyze(snapshotID int64, jumpThresholdNs int64) (*AnalyzeResult, error) {
	snap, err := a.DB.GetSnapshot(snapshotID)
	if err != nil {
		return nil, err
	}
	if snap.Status == model.SnapshotCollecting {
		if _, err := a.Topology.LockSnapshot(snapshotID); err != nil {
			return nil, err
		}
	}

	summary := fmt.Sprintf("clock drift localization for snapshot %q", snap.Name)
	ev, err := a.DB.InsertEvent(snapshotID, summary)
	if err != nil {
		return nil, err
	}

	cands, err := a.Diagnosis.Diagnose(ev.ID, jumpThresholdNs)
	if err != nil {
		return nil, err
	}

	propRes, err := a.Propagation.Analyze(snapshotID, jumpThresholdNs)
	if err != nil {
		return nil, err
	}

	ev, err = a.DB.GetEvent(ev.ID)
	if err != nil {
		return nil, err
	}
	return &AnalyzeResult{Event: ev, Candidates: cands, Propagation: propRes}, nil
}

// ListCandidates 列出事件候选（按分数降序）。
func (a *App) ListCandidates(eventID int64) ([]*model.RootCauseCandidate, error) {
	return a.DB.ListCandidatesByEvent(eventID)
}

// EvidencePaths 列出候选的证据路径。
func (a *App) EvidencePaths(candidateID int64) ([]*model.EvidencePath, error) {
	return a.DB.ListEvidencePaths(candidateID)
}
