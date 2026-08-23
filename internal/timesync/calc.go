// Package timesync 提供时钟同步测量计算与单位换算。
package timesync

import "math"

// OffsetFromTimestamps 由 NTP 四个时间戳计算时钟偏移（θ）。
//
//	t0: 客户端发送时刻；t1: 服务端接收时刻；
//	t2: 服务端发送时刻；t3: 客户端接收时刻。
//
// 偏移 θ = ((t1 - t0) + (t2 - t3)) / 2，单位与输入一致。
// 往返延迟 δ = (t3 - t0) - (t2 - t1)。
func OffsetFromTimestamps(t0, t1, t2, t3 int64) (offset int64, roundtrip int64) {
	offset = ((t1 - t0) + (t2 - t3)) / 2
	roundtrip = (t3 - t0) - (t2 - t1)
	return offset, roundtrip
}

// OffsetFromPair 由一次对称测量（上行延迟 dUp、下行延迟 dDown）计算偏移。
// 适用于 PTP 场景：offset = (dDown - dUp) / 2。
func OffsetFromPair(dUp, dDown int64) int64 {
	return (dDown - dUp) / 2
}

// RoundtripDelay 计算往返延迟：δ = (t3 - t0) - (t2 - t1)。
func RoundtripDelay(t0, t1, t2, t3 int64) int64 {
	return (t3 - t0) - (t2 - t1)
}

// AllanDeviation 计算一组频率偏差样本的艾伦方差平方根（σ_y(τ)），
// 用于量化节点时钟的短期稳定性，作为根因候选评分依据之一。
// 输入 freqs 为相邻采样间隔内的归一化频率偏差（无量纲，如 ppm）。
func AllanDeviation(freqs []float64) float64 {
	if len(freqs) < 2 {
		return 0
	}
	n := len(freqs)
	var sum float64
	for i := 0; i < n-1; i++ {
		d := freqs[i+1] - freqs[i]
		sum += d * d
	}
	return math.Sqrt(sum / float64(2*(n-1)))
}

// JumpMagnitude 计算相邻两个偏移值之间的跳变幅度（绝对值）。
func JumpMagnitude(prev, curr int64) int64 {
	d := curr - prev
	if d < 0 {
		return -d
	}
	return d
}

// RelativeSkew 计算两个节点偏移序列的相对偏斜度，
// 用于判断下游节点是否跟随上游节点跳变（传播判定）。
// 返回值越接近 1 表示下游越紧跟上上游变化。
func RelativeSkew(upstream []int64, downstream []int64) float64 {
	if len(upstream) < 2 || len(downstream) < 2 {
		return 0
	}
	n := min(len(upstream), len(downstream))
	var num, denA, denB float64
	var upMean, downMean float64
	for i := 0; i < n; i++ {
		upMean += float64(upstream[i])
		downMean += float64(downstream[i])
	}
	upMean /= float64(n)
	downMean /= float64(n)
	for i := 0; i < n; i++ {
		ua := float64(upstream[i]) - upMean
		da := float64(downstream[i]) - downMean
		num += ua * da
		denA += ua * ua
		denB += da * da
	}
	if denA == 0 || denB == 0 {
		return 0
	}
	return num / (math.Sqrt(denA) * math.Sqrt(denB))
}
