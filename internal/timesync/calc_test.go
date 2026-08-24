package timesync

import "testing"

func TestOffsetFromTimestamps(t *testing.T) {
	// 对称延迟：t1-t0 = t3-t2 = 100ns，offset 应为 0。
	offset, rtt := OffsetFromTimestamps(1000, 1100, 1100, 1200)
	if offset != 0 {
		t.Fatalf("expected offset 0, got %d", offset)
	}
	if rtt != 200 {
		t.Fatalf("expected rtt 200, got %d", rtt)
	}
}

func TestOffsetFromTimestampsSkewed(t *testing.T) {
	// t0=0, t1=50, t2=50, t3=100：上行 50ns、下行 50ns，offset=0，rtt=100。
	offset, rtt := OffsetFromTimestamps(0, 50, 50, 100)
	if offset != 0 || rtt != 100 {
		t.Fatalf("unexpected offset=%d rtt=%d", offset, rtt)
	}
}

func TestRoundtripDelay(t *testing.T) {
	// δ = (t3-t0) - (t2-t1) = (40-0) - (20-10) = 30。
	if d := RoundtripDelay(0, 10, 20, 40); d != 30 {
		t.Fatalf("expected 30, got %d", d)
	}
}

func TestToNanos(t *testing.T) {
	cases := []struct {
		value int64
		unit  Unit
		want  int64
	}{
		{1, UnitSecond, 1_000_000_000},
		{5, UnitMilli, 5_000_000},
		{1000, UnitMicro, 1_000_000},
		{123, UnitNano, 123},
	}
	for _, c := range cases {
		got, err := ToNanos(c.value, c.unit)
		if err != nil {
			t.Fatalf("ToNanos(%d,%s): %v", c.value, c.unit, err)
		}
		if got != c.want {
			t.Fatalf("ToNanos(%d,%s)=%d want %d", c.value, c.unit, got, c.want)
		}
	}
}

func TestParseUnit(t *testing.T) {
	if u, err := ParseUnit(""); err != nil || u != UnitNano {
		t.Fatalf("empty unit should default to ns, got %q err=%v", u, err)
	}
	if _, err := ParseUnit("foobar"); err == nil {
		t.Fatal("expected error for bad unit")
	}
}

func TestFormatNanos(t *testing.T) {
	cases := map[int64]string{
		0:            "0ns",
		1500:         "1.5us",
		5_000_000:    "5ms",
		1_500_000_000: "1.5s",
	}
	for ns, want := range cases {
		if got := FormatNanos(ns); got != want {
			t.Fatalf("FormatNanos(%d)=%q want %q", ns, got, want)
		}
	}
}

func TestJumpMagnitude(t *testing.T) {
	if JumpMagnitude(10, 20) != 10 {
		t.Fatal("expected 10")
	}
	if JumpMagnitude(20, 5) != 15 {
		t.Fatal("expected 15")
	}
}

func TestAllanDeviation(t *testing.T) {
	if AllanDeviation([]float64{1, 1, 1, 1}) != 0 {
		t.Fatal("constant series should have zero Allan deviation")
	}
	if AllanDeviation([]float64{1, 3, 1, 3}) <= 0 {
		t.Fatal("varying series should have positive Allan deviation")
	}
}

func TestRelativeSkew(t *testing.T) {
	up := []int64{0, 100, 200, 300}
	down := []int64{10, 110, 210, 310}
	if sk := RelativeSkew(up, down); sk < 0.99 {
		t.Fatalf("expected skew ~1, got %f", sk)
	}
}
