package main

import "math"

// 采样与目标常量。默认 30 天、1 分钟粒度。
const (
	DefaultDays     = 30
	MinutesPerDay   = 1440
	MinutesInMonth  = DefaultDays * MinutesPerDay // 43200
	SLOTarget       = 0.999                       // 99.9%
	BadMinuteRate   = 0.01                        // 失败率超过 1% 视为坏分钟
	BadLatencyMs    = 50.0                        // P95 时延超过 50ms 视为坏
	DayTrafficReqPS = 800.0
	NightTrafficRPS = 100.0
)

// Incident 是一次人为注入的故障。
type Incident struct {
	FromMin int
	ToMin   int
	BadRate float64 // 期间失败率
	LatMs   float64 // 期间 P95 时延
	Note    string
}

// Incidents 是 30 天里注入的四次事故，故意分布在夜与昼。
//
// 分布是设计过的：夜间事故在时间口径上影响很大、在事件口径上影响很小，
// 白天事故反过来。三种口径的分歧就来自这里。
var Incidents = []Incident{
	{FromMin: 3*MinutesPerDay + 2*60, ToMin: 3*MinutesPerDay + 2*60 + 60, BadRate: 0.30, LatMs: 220, Note: "夜间大事故（劣化一小时才被发现）"},
	{FromMin: 12*MinutesPerDay + 14*60, ToMin: 12*MinutesPerDay + 14*60 + 15, BadRate: 0.25, LatMs: 180, Note: "白天大事故（PFC 风暴）"},
	{FromMin: 20*MinutesPerDay + 10*60, ToMin: 20*MinutesPerDay + 10*60 + 5, BadRate: 0.06, LatMs: 40, Note: "白天小抖动"},
	{FromMin: 27*MinutesPerDay + 3*60, ToMin: 27*MinutesPerDay + 3*60 + 5, BadRate: 0.05, LatMs: 35, Note: "夜间小抖动"},
	{FromMin: MinutesInMonth - 10, ToMin: MinutesInMonth, BadRate: 0.20, LatMs: 150, Note: "正在发生（数据窗口末尾）"},
}

// Sample 是一分钟的聚合指标。
type Sample struct {
	TsMin int     // 相对窗口起点的分钟数
	Reqs  int     // 该分钟请求数（也是事件口径的权重）
	Bad   int     // 失败请求数
	LatMs float64 // 该分钟 P95 时延
}

// FailRate 返回该分钟失败率。
func (s Sample) FailRate() float64 {
	if s.Reqs == 0 {
		return 0
	}
	return float64(s.Bad) / float64(s.Reqs)
}

// Bad 表示这一分钟在时间口径下算不算坏。
func (s Sample) BadMinute() bool {
	return s.FailRate() > BadMinuteRate || s.LatMs > BadLatencyMs
}

// trafficRate 返回某一分钟的基础流量（请求/秒），带昼夜起伏。
func trafficRate(tsMin int) float64 {
	hour := float64(tsMin%MinutesPerDay) / 60
	mid := (DayTrafficReqPS + NightTrafficRPS) / 2
	amp := (DayTrafficReqPS - NightTrafficRPS) / 2
	return mid + amp*math.Sin(2*math.Pi*(hour-9)/24)
}

// Generate 造 30 天、1 分钟粒度的指标流。
func Generate(days int) []Sample {
	if days <= 0 {
		days = DefaultDays
	}
	total := days * MinutesPerDay
	out := make([]Sample, 0, total)

	for t := 0; t < total; t++ {
		rate := trafficRate(t)
		reqs := int(rate * 60)
		if reqs < 1 {
			reqs = 1
		}
		badRate := 0.0005 // 基线失败率
		lat := 8 + 2*math.Sin(float64(t)) // 基线 P95 时延

		for _, in := range Incidents {
			if t >= in.FromMin && t < in.ToMin {
				badRate = in.BadRate
				lat = in.LatMs
				break
			}
		}

		bad := int(float64(reqs) * badRate)
		out = append(out, Sample{TsMin: t, Reqs: reqs, Bad: bad, LatMs: round1(lat)})
	}
	return out
}

// round1 保留一位小数。
func round1(v float64) float64 { return math.Round(v*10) / 10 }

// IncidentAt 返回覆盖该分钟的事故，没有则返回 nil。
func IncidentAt(tsMin int) *Incident {
	for i := range Incidents {
		if tsMin >= Incidents[i].FromMin && tsMin < Incidents[i].ToMin {
			return &Incidents[i]
		}
	}
	return nil
}
