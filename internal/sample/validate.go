package sample

import (
	"task210-clocktrace/internal/timesync"
)

// ValidateUnit 校验并规范化单位字符串；空串按纳秒处理。
func ValidateUnit(unit string) (timesync.Unit, error) {
	return timesync.ParseUnit(unit)
}

// ValidateOffsetRange 判断偏移换算成纳秒后是否可表示。
func ValidateOffsetRange(offsetNs int64) bool {
	return offsetNs <= maxOffsetNs && offsetNs >= -maxOffsetNs
}

// ValidateRoundtrip 判断往返延迟是否合法（非负）。
func ValidateRoundtrip(roundtripNs int64) bool {
	return roundtripNs >= 0
}

// AnomalyReason 描述样本校验失败的原因，供错误映射使用。
type AnomalyReason string

const (
	ReasonBadUnit     AnomalyReason = "bad_unit"
	ReasonOverflow    AnomalyReason = "offset_overflow"
	ReasonNegativeRTT AnomalyReason = "negative_roundtrip"
	ReasonUnknownNode AnomalyReason = "unknown_node"
)
