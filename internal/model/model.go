// Package model 定义网络时间同步根因定位服务的核心实体。
package model

import "time"

// Snapshot 网络快照：一次网络拓扑与时钟同步层级采集的完整视图。
type Snapshot struct {
	ID         int64      `json:"id"`
	Name       string     `json:"name"`
	Status     string     `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
	ArchivedAt *time.Time `json:"archived_at,omitempty"`
}

// Node 拓扑节点：参与时间同步的网络设备。
type Node struct {
	ID         int64  `json:"id"`
	SnapshotID int64  `json:"snapshot_id"`
	NodeKey    string `json:"node_key"` // 快照内唯一键
	Role       string `json:"role"`     // grandmaster / boundary / ordinary
	Hostname   string `json:"hostname"`
	ClockClass int    `json:"clock_class"`
}

// Link 拓扑边：时间同步层级关系（上游 -> 下游）。
type Link struct {
	ID         int64  `json:"id"`
	SnapshotID int64  `json:"snapshot_id"`
	FromNode   string `json:"from_node"` // 上游节点
	ToNode     string `json:"to_node"`   // 下游节点
	Protocol   string `json:"protocol"`  // ntp / ptp
	Declared   bool   `json:"declared"`  // 是否显式声明（环检测依据）
}

// Sample 节点样本：一次时钟同步测量。
type Sample struct {
	ID          int64     `json:"id"`
	SnapshotID  int64     `json:"snapshot_id"`
	NodeKey     string    `json:"node_key"`
	Sequence    int64     `json:"sequence"`     // 幂等键：同一节点同一序号只入库一次
	OffsetNs    int64     `json:"offset_ns"`    // 时钟偏移（统一纳秒）
	RoundtripNs int64     `json:"roundtrip_ns"` // 往返延迟（统一纳秒）
	Unit        string    `json:"unit"`         // 上报原始单位 ns/us/ms/s
	SourceID    string    `json:"source_id"`    // 时钟源标识
	Status      string    `json:"status"`
	CollectedAt time.Time `json:"collected_at"`
}

// DriftEvent 漂移事件：一次时钟漂移的诊断单元。
type DriftEvent struct {
	ID         int64      `json:"id"`
	SnapshotID int64      `json:"snapshot_id"`
	Status     string     `json:"status"`
	Summary    string     `json:"summary"`
	CreatedAt  time.Time  `json:"created_at"`
	SealedAt   *time.Time `json:"sealed_at,omitempty"`
	Revision   int        `json:"revision"`
}

// RootCauseCandidate 根因候选：沿拓扑推导出的异常传播源头。
type RootCauseCandidate struct {
	ID       int64   `json:"id"`
	EventID  int64   `json:"event_id"`
	Kind     string  `json:"kind"` // source_switch / link_anomaly / upstream_jump
	NodeKey  string  `json:"node_key"`
	LinkID   int64   `json:"link_id,omitempty"`
	Score    float64 `json:"score"`
	Status   string  `json:"status"`
	Evidence string  `json:"evidence"`
}

// EvidencePath 证据路径：根因节点沿拓扑下游的受影响节点链。
type EvidencePath struct {
	ID          int64  `json:"id"`
	CandidateID int64  `json:"candidate_id"`
	OrderIdx    int    `json:"order_idx"`
	NodeKey     string `json:"node_key"`
	OffsetNs    int64  `json:"offset_ns"`
}

// Verdict 裁决记录：工程师对候选或样本做出的确认 / 否决 / 标记不可信。
type Verdict struct {
	ID          int64     `json:"id"`
	EventID     int64     `json:"event_id"`
	CandidateID int64     `json:"candidate_id"`
	Action      string    `json:"action"` // confirm / reject / mark_untrusted
	Note        string    `json:"note"`
	CreatedAt   time.Time `json:"created_at"`
}
