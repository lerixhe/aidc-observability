package main

import "math"

// 采样与阈值常量。默认窗口 1 小时，秒级采样。
const (
	DefaultWindowSec = 3600
	SampleInterval   = 1 // 秒
	// QueueThresholdKB 是判读用的告警阈值。
	QueueThresholdKB = 400.0
	// DangerWindowSec 是人为注入的劣化窗口，报告里的「事故时段」用它。
	DangerFromSec = 2400 // 第 40 分钟
	DangerToSec   = 3000 // 第 50 分钟
)

// Point 是一个秒级采样点。
type Point struct {
	TsSec int     // 相对窗口起点的秒数
	KB    float64 // 出口队列占用
}

// Generate 造一段 1 小时的队列序列。
//
// 三个成分叠加：
//  1. 基线：20 KB 上下，带 5 分钟周期的缓慢起伏
//  2. 微突发：每 5 分钟一次，持续 2 秒，冲到 180 KB（秒级可见，分钟聚合会被摊平）
//  3. 劣化窗口：第 40 到 50 分钟，波形在 180 KB 与 520 KB 之间来回
func Generate(windowSec int) []Point {
	if windowSec <= 0 {
		windowSec = DefaultWindowSec
	}
	pts := make([]Point, 0, windowSec/SampleInterval)
	for t := 0; t < windowSec; t += SampleInterval {
		kb := 20 + 5*math.Sin(2*math.Pi*float64(t)/300)

		// 微突发：每个 5 分钟周期的最后 2 秒
		if t%300 >= 298 {
			kb += 160
		}

		// 劣化窗口：多数时间温和，短时冲顶
		if t >= DangerFromSec && t < DangerToSec {
			phase := (t - DangerFromSec) % 30
			if phase < 3 {
				kb = 520
			} else {
				kb = 180 + 20*math.Sin(2*math.Pi*float64(phase)/30)
			}
		}

		pts = append(pts, Point{TsSec: t, KB: round1(kb)})
	}
	return pts
}

// round1 保留一位小数，让输出稳定、便于 expected/ 比对。
func round1(v float64) float64 {
	return math.Round(v*10) / 10
}

// DangerPoints 返回落在劣化窗口内的点。
func DangerPoints(pts []Point) []Point {
	out := make([]Point, 0, len(pts))
	for _, p := range pts {
		if p.TsSec >= DangerFromSec && p.TsSec < DangerToSec {
			out = append(out, p)
		}
	}
	return out
}
