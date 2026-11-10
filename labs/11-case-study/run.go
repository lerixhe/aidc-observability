package main

import "fmt"

// 执行一条路径，并把耗时按 MTTR 五段累加。
//
// 五段拆开算，是为了看清「时间花在哪」：同样是 40 分钟，
// 花在缩小上和在确认上是两种不同的组织能力问题。

// PhaseCost 是某一阶段的耗时汇总。
type PhaseCost struct {
	Phase   Phase
	Minutes float64
	Steps   []int
}

// Run 是一条路径的执行结果。
type Run struct {
	Path   Path
	Total  float64
	Phases []PhaseCost
	Units  int // 实际执行的步骤数
}

// Execute 按顺序走完一条路径。
func Execute(p Path) Run {
	r := Run{Path: p, Units: len(p.Steps)}
	idx := map[Phase]int{}
	for _, ph := range PhaseOrder {
		idx[ph] = len(r.Phases)
		r.Phases = append(r.Phases, PhaseCost{Phase: ph})
	}
	for _, s := range p.Steps {
		i, ok := idx[s.Phase]
		if !ok {
			continue
		}
		r.Phases[i].Minutes += s.Cost
		r.Phases[i].Steps = append(r.Phases[i].Steps, s.ID)
		r.Total += s.Cost
	}
	return r
}

// Share 返回某阶段占总时间的比例。
func (r Run) Share(p Phase) float64 {
	if r.Total == 0 {
		return 0
	}
	for _, pc := range r.Phases {
		if pc.Phase == p {
			return pc.Minutes / r.Total
		}
	}
	return 0
}

// Verdict 给一条路径一句评价，用于对比表。
func (r Run) Verdict() string {
	switch r.Path.Name {
	case "best":
		return "一次命中，没有返工"
	case "naive":
		return "换错一件模块，返工两轮"
	}
	return ""
}

// fmtMin 把分钟数打成「1.5」这种一位小数，整数时不带小数点。
func fmtMin(m float64) string {
	if m == float64(int(m)) {
		return fmt.Sprintf("%.0f", m)
	}
	return fmt.Sprintf("%.1f", m)
}
