package timesync

import (
	"fmt"
	"math"
	"strings"

	"task210-clocktrace/internal/model"
)

// Unit 时钟偏移/延迟的单位。
type Unit string

const (
	UnitNano       Unit = "ns"
	UnitMicro      Unit = "us"
	UnitMilli      Unit = "ms"
	UnitSecond     Unit = "s"
	UnitUnspecified Unit = ""
)

// unitScale 各单位到纳秒的换算系数。
var unitScale = map[Unit]int64{
	UnitNano:   1,
	UnitMicro:  1000,
	UnitMilli:  1000_000,
	UnitSecond: 1000_000_000,
}

// ParseUnit 解析单位字符串；空串缺省按纳秒；非法单位返回 ErrUnitMismatch。
func ParseUnit(s string) (Unit, error) {
	u := Unit(strings.ToLower(strings.TrimSpace(s)))
	if u == UnitUnspecified {
		return UnitNano, nil // 缺省按纳秒
	}
	if _, ok := unitScale[u]; !ok {
		return UnitUnspecified, fmt.Errorf("%w: %q", model.ErrUnitMismatch, s)
	}
	return u, nil
}

// ToNanos 将 value（以 u 为单位）换算为纳秒；溢出时返回 ErrOffsetOverflow。
func ToNanos(value int64, u Unit) (int64, error) {
	scale, ok := unitScale[u]
	if !ok {
		return 0, fmt.Errorf("%w: %q", model.ErrUnitMismatch, string(u))
	}
	if value > math.MaxInt64/scale || value < math.MinInt64/scale {
		return 0, model.ErrOffsetOverflow
	}
	return value * scale, nil
}

// FormatNanos 将纳秒值格式化为人可读的字符串（选择合适单位）。
func FormatNanos(ns int64) string {
	abs := ns
	if abs < 0 {
		abs = -abs
	}
	switch {
	case abs >= 1000_000_000:
		return formatScaled(ns, 1000_000_000, "s")
	case abs >= 1000_000:
		return formatScaled(ns, 1000_000, "ms")
	case abs >= 1000:
		return formatScaled(ns, 1000, "us")
	default:
		return formatScaled(ns, 1, "ns")
	}
}

func formatScaled(ns, scale int64, unit string) string {
	whole := ns / scale
	frac := ns % scale
	if frac == 0 {
		return itoa(whole) + unit
	}
	if frac < 0 {
		frac = -frac
	}
	digits := 0
	for s := scale; s > 1; s /= 10 {
		digits++
	}
	f := itoa(frac)
	for len(f) < digits {
		f = "0" + f
	}
	f = strings.TrimRight(f, "0")
	return itoa(whole) + "." + f + unit
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}