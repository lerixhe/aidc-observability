package main

import "fmt"

// Caliber 是一种 SLO 口径。同一份数据，三种口径给出三个数。
type Caliber string

const (
	// CaliberTime 时间口径：坏分钟占总分钟比例。
	CaliberTime Caliber = "时间口径"
	// CaliberEvent 事件口径：坏请求占总请求比例，按流量加权。
	CaliberEvent Caliber = "事件口径"
	// CaliberLatency 分位口径：P95 时延达标的分钟占比。
	CaliberLatency Caliber = "分位口径"
)

// Calibers 是报告里的展示顺序。
var Calibers = []Caliber{CaliberTime, CaliberEvent, CaliberLatency}

// Result 是一种口径的计算结果。
type Result struct {
	Caliber      Caliber
	Availability float64 // 0 到 1
	BadUnits     int     // 坏分钟数或坏请求数
	TotalUnits   int     // 总分钟数或总请求数
	Met          bool
	Note         string
}

// Compute 按指定口径算可用率。
func Compute(c Caliber, samples []Sample, slo float64) Result {
	r := Result{Caliber: c}
	switch c {
	case CaliberTime:
		for _, s := range samples {
			r.TotalUnits++
			if s.BadMinute() {
				r.BadUnits++
			}
		}
		r.Note = "分母是分钟数，夜间小流量事故与白天大流量事故权重相同"
	case CaliberEvent:
		for _, s := range samples {
			r.TotalUnits += s.Reqs
			r.BadUnits += s.Bad
		}
		r.Note = "分母是请求数，夜间事故被低流量稀释"
	case CaliberLatency:
		for _, s := range samples {
			r.TotalUnits++
			if s.LatMs <= BadLatencyMs {
				r.BadUnits++ // 这里 BadUnits 记的是「达标分钟数」，见下面 Availability 的算法
			}
		}
		if r.TotalUnits > 0 {
			r.Availability = float64(r.BadUnits) / float64(r.TotalUnits)
		}
		r.BadUnits = r.TotalUnits - r.BadUnits
		r.Met = r.Availability >= slo
		r.Note = "只看时延，完全忽略失败率：失败快也是「达标」"
		return r
	}

	if r.TotalUnits > 0 {
		r.Availability = 1 - float64(r.BadUnits)/float64(r.TotalUnits)
	}
	r.Met = r.Availability >= slo
	return r
}

// GapBp 返回可用率与目标之间差多少个基点（万分之一）。
func (r Result) GapBp(slo float64) float64 {
	return (r.Availability - slo) * 10000
}

// Verdict 返回达标判定的一句话。
func (r Result) Verdict() string {
	if r.Met {
		return "达标"
	}
	return "未达标"
}

// Compare 把三种口径摆在一起，指出分歧最大的两个。
func Compare(rs []Result) string {
	if len(rs) < 2 {
		return "口径不足，无法对比"
	}
	hi, lo := rs[0], rs[0]
	for _, r := range rs[1:] {
		if r.Availability > hi.Availability {
			hi = r
		}
		if r.Availability < lo.Availability {
			lo = r
		}
	}
	return fmt.Sprintf("最宽 %s %.3f%% 与最严 %s %.3f%% 相差 %.0f 个基点",
		hi.Caliber, hi.Availability*100, lo.Caliber, lo.Availability*100,
		(hi.Availability-lo.Availability)*10000)
}
