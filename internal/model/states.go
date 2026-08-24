package model

// 快照状态。
const (
	SnapshotCollecting = "collecting" // 收集拓扑中，可增删节点/边
	SnapshotAnalyzable = "analyzable" // 已锁定，可接收样本并分析
	SnapshotArchived   = "archived"   // 已归档，只读
)

// 样本状态。
const (
	SamplePending   = "pending"   // 待校验
	SampleValid     = "valid"     // 校验通过
	SampleAnomaly   = "anomaly"   // 校验异常（单位错误 / 负延迟 / 偏移溢出）
	SampleDuplicate = "duplicate" // 重复样本（幂等命中）
)

// 漂移事件状态。
const (
	EventObserved             = "observed"              // 已观察（收到异常样本）
	EventLocalizing           = "localizing"            // 定位中（已生成候选）
	EventInsufficientEvidence = "insufficient_evidence" // 证据不足（无候选）
	EventConfirmed            = "confirmed"             // 根因已确认
	EventSealed               = "sealed"                // 已封存，只读
)

// 根因候选状态。
const (
	CandidateGenerated          = "generated"           // 已生成
	CandidatePendingConfirmation = "pending_confirmation" // 待确认
	CandidateRejected           = "rejected"            // 已否决
	CandidateConfirmed          = "confirmed"           // 已确认
)

// 候选根因类型。
const (
	CauseSourceSwitch = "source_switch" // 时钟源切换导致跳变
	CauseLinkAnomaly  = "link_anomaly"  // 同步链路异常
	CauseUpstreamJump = "upstream_jump" // 上游跳变传播
)

// 裁决动作。
const (
	ActionConfirm       = "confirm"
	ActionReject        = "reject"
	ActionMarkUntrusted = "mark_untrusted"
	ActionSeal          = "seal"
)

// eventTransitions 事件状态机邻接表。
var eventTransitions = map[string][]string{
	EventObserved:             {EventLocalizing},
	EventLocalizing:           {EventInsufficientEvidence, EventConfirmed},
	EventInsufficientEvidence: {EventLocalizing},
	EventConfirmed:            {EventSealed},
	EventSealed:               {},
}

// CanTransitionEvent 判断事件状态是否可从 from 流转到 to。
func CanTransitionEvent(from, to string) bool {
	if from == "" {
		return true
	}
	for _, t := range eventTransitions[from] {
		if t == to {
			return true
		}
	}
	return false
}

// candidateTransitions 候选状态机邻接表。
var candidateTransitions = map[string][]string{
	CandidateGenerated:           {CandidatePendingConfirmation},
	CandidatePendingConfirmation: {CandidateRejected, CandidateConfirmed},
	CandidateRejected:            {},
	CandidateConfirmed:           {},
}

// CanTransitionCandidate 判断候选状态是否可从 from 流转到 to。
func CanTransitionCandidate(from, to string) bool {
	if from == "" {
		return true
	}
	for _, t := range candidateTransitions[from] {
		if t == to {
			return true
		}
	}
	return false
}
