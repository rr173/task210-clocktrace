// Package smoke 实现 --smoke-test 端到端自检：真实创建拓扑、提交样本、
// 触发根因定位、否决/确认候选、封存事件，关闭并重新打开数据库验证
// 持久化与重启恢复，最后以 0 退出码结束。
package smoke

import (
	"fmt"
	"os"
	"time"

	"task210-clocktrace/internal/model"
	"task210-clocktrace/internal/service"
	"task210-clocktrace/internal/store"
)

// Main 自检入口：args[0] 为数据库路径。
func Main(args []string) {
	dbPath := "clocktrace.db"
	if len(args) > 0 && args[0] != "" {
		dbPath = args[0]
	}
	if err := Run(dbPath); err != nil {
		fmt.Fprintf(os.Stderr, "smoke test FAILED: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("smoke test PASSED")
}

// Run 执行端到端自检。返回 nil 表示全部通过。
func Run(dbPath string) error {
	step := func(n int, name string, fn func() error) error {
		fmt.Printf("[smoke %d/9] %s ...\n", n, name)
		if err := fn(); err != nil {
			return fmt.Errorf("smoke step %d (%s): %w", n, name, err)
		}
		return nil
	}

	var snapshotID int64
	var eventID int64
	var confirmedCandidateID int64

	// 第一步：创建快照与拓扑。
	if err := step(1, "创建快照与同步拓扑", func() error {
		db, err := store.Open(dbPath)
		if err != nil {
			return err
		}
		defer db.Close()
		app := service.New(db)

		snap, err := app.Topology.CreateSnapshot("smoke-network")
		if err != nil {
			return err
		}
		snapshotID = snap.ID

		nodes := []struct{ key, role, host string }{
			{"source", "grandmaster", "gm-1"},
			{"edge1", "boundary", "bc-1"},
			{"edge2", "boundary", "bc-2"},
			{"leaf", "ordinary", "oc-1"},
		}
		for _, n := range nodes {
			if _, err := app.Topology.AddNode(snapshotID, n.key, n.role, n.host, 6); err != nil {
				return err
			}
		}
		edges := []struct{ from, to string }{
			{"source", "edge1"},
			{"edge1", "edge2"},
			{"edge2", "leaf"},
		}
		for _, e := range edges {
			if _, err := app.Topology.AddLink(snapshotID, e.from, e.to, "ntp", true); err != nil {
				return err
			}
		}
		if _, err := app.Topology.LockSnapshot(snapshotID); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	// 第二步：提交样本（上游时钟源跳变 5ms + edge1 源切换）。
	if err := step(2, "提交节点同步样本", func() error {
		db, err := store.Open(dbPath)
		if err != nil {
			return err
		}
		defer db.Close()
		app := service.New(db)

		type sampleSpec struct {
			node   string
			seq    int64
			offset int64
			src    string
		}
		specs := []sampleSpec{
			{"source", 1, 0, "gm-a"},
			{"source", 2, 5_000_000, "gm-a"},
			{"edge1", 1, 0, "gm-a"},
			{"edge1", 2, 5_100_000, "gm-b"},
			{"edge2", 1, 0, "gm-a"},
			{"edge2", 2, 5_200_000, "gm-a"},
			{"leaf", 1, 0, "gm-a"},
			{"leaf", 2, 5_300_000, "gm-a"},
		}
		for _, sp := range specs {
			if _, err := app.Samples.Submit(snapshotID, sp.node, sp.seq, sp.offset, 100_000, "ns", sp.src, time.Now().UTC()); err != nil {
				return err
			}
		}
		// 幂等验证：重复提交同一序号返回已有样本，不报错。
		if _, err := app.Samples.Submit(snapshotID, "source", 1, 0, 100_000, "ns", "gm-a", time.Now().UTC()); err != nil {
			return fmt.Errorf("idempotent resubmit: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}

	// 第三步：分析并生成候选。
	if err := step(3, "触发根因定位", func() error {
		db, err := store.Open(dbPath)
		if err != nil {
			return err
		}
		defer db.Close()
		app := service.New(db)

		res, err := app.Analyze(snapshotID, 1_000_000)
		if err != nil {
			return err
		}
		eventID = res.Event.ID
		if res.Event.Status != model.EventLocalizing {
			return fmt.Errorf("expected event localizing, got %s", res.Event.Status)
		}
		if len(res.Candidates) < 2 {
			return fmt.Errorf("expected >= 2 candidates, got %d", len(res.Candidates))
		}
		return nil
	}); err != nil {
		return err
	}

	// 第四步：否决源切换候选（缺样本路径）。
	if err := step(4, "否决源切换候选", func() error {
		db, err := store.Open(dbPath)
		if err != nil {
			return err
		}
		defer db.Close()
		app := service.New(db)

		cands, err := app.ListCandidates(eventID)
		if err != nil {
			return err
		}
		var switchCand *model.RootCauseCandidate
		for _, c := range cands {
			if c.Kind == model.CauseSourceSwitch {
				switchCand = c
				break
			}
		}
		if switchCand == nil {
			return fmt.Errorf("expected a source_switch candidate")
		}
		if err := app.Verdict.Reject(eventID, switchCand.ID, "insufficient sample path"); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	// 第五步：确认上游跳变候选。
	if err := step(5, "确认上游跳变候选", func() error {
		db, err := store.Open(dbPath)
		if err != nil {
			return err
		}
		defer db.Close()
		app := service.New(db)

		cands, err := app.ListCandidates(eventID)
		if err != nil {
			return err
		}
		var jumpCand *model.RootCauseCandidate
		for _, c := range cands {
			if c.Kind == model.CauseUpstreamJump && c.Status == model.CandidatePendingConfirmation {
				jumpCand = c
				break
			}
		}
		if jumpCand == nil {
			return fmt.Errorf("expected an upstream_jump candidate")
		}
		if jumpCand.NodeKey != "source" {
			return fmt.Errorf("expected earliest node 'source', got %s", jumpCand.NodeKey)
		}
		confirmedCandidateID = jumpCand.ID
		if err := app.Verdict.Confirm(eventID, jumpCand.ID, "upstream clock source jump"); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	// 第六步：封存事件。
	if err := step(6, "封存漂移事件", func() error {
		db, err := store.Open(dbPath)
		if err != nil {
			return err
		}
		defer db.Close()
		app := service.New(db)
		if err := app.Verdict.Seal(eventID, "root cause confirmed and archived"); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	// 第七步：关闭后重开验证恢复。
	if err := step(7, "重开数据库验证恢复", func() error {
		db, err := store.Open(dbPath)
		if err != nil {
			return err
		}
		defer db.Close()
		app := service.New(db)

		ev, err := app.DB.GetEvent(eventID)
		if err != nil {
			return err
		}
		if ev.Status != model.EventSealed {
			return fmt.Errorf("expected sealed event after reopen, got %s", ev.Status)
		}
		c, err := app.DB.GetCandidate(confirmedCandidateID)
		if err != nil {
			return err
		}
		if c.Status != model.CandidateConfirmed {
			return fmt.Errorf("expected confirmed candidate after reopen, got %s", c.Status)
		}
		paths, err := app.EvidencePaths(confirmedCandidateID)
		if err != nil {
			return err
		}
		if len(paths) == 0 {
			return fmt.Errorf("expected non-empty evidence path after reopen")
		}
		samples, err := app.Samples.ListBySnapshot(snapshotID)
		if err != nil {
			return err
		}
		if len(samples) != 8 {
			return fmt.Errorf("expected 8 samples after reopen, got %d", len(samples))
		}
		return nil
	}); err != nil {
		return err
	}

	// 第八步：验证封存事件不可修改。
	if err := step(8, "验证封存事件只读", func() error {
		db, err := store.Open(dbPath)
		if err != nil {
			return err
		}
		defer db.Close()
		app := service.New(db)
		if err := app.Verdict.Seal(eventID, "attempt re-seal"); err == nil {
			return fmt.Errorf("expected re-seal to fail on sealed event")
		} else if err != model.ErrSealed {
			return fmt.Errorf("expected ErrSealed, got %v", err)
		}
		return nil
	}); err != nil {
		return err
	}

	// 第九步：统计一致性。
	if err := step(9, "统计一致性", func() error {
		db, err := store.Open(dbPath)
		if err != nil {
			return err
		}
		defer db.Close()
		app := service.New(db)
		stats, err := app.ComputeStats()
		if err != nil {
			return err
		}
		if stats.Snapshots < 1 || stats.Nodes != 4 || stats.Samples != 8 {
			return fmt.Errorf("unexpected stats: %+v", stats)
		}
		if stats.SealedEvents < 1 || stats.Confirmed < 1 {
			return fmt.Errorf("expected sealed event and confirmed candidate: %+v", stats)
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}
