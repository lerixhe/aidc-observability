package main

import (
	"fmt"
	"io"
	"math/rand"
	"sort"
	"strings"
	"time"
)

// MethodResult 是一个方法的命中率统计。
type MethodResult struct {
	Name    string
	Top1    float64
	Top3    float64
	AvgRank float64
	ranks   []int
}

func (m *MethodResult) add(rank int) {
	m.ranks = append(m.ranks, rank)
}

func (m *MethodResult) finish() {
	if len(m.ranks) == 0 {
		return
	}
	top1, top3, sum := 0, 0, 0
	for _, r := range m.ranks {
		if r == 1 {
			top1++
		}
		if r <= 3 {
			top3++
		}
		sum += r
	}
	n := float64(len(m.ranks))
	m.Top1 = float64(top1) / n
	m.Top3 = float64(top3) / n
	m.AvgRank = float64(sum) / n
}

// RoundDetail 是一轮的完整过程，用来在报告里展示「算法到底做了什么」。
type RoundDetail struct {
	Truth      Truth
	Alerts     []Alert
	Incidents  []Incident
	MainInc    Incident
	Graph      []Candidate
	ByCount    []string
	ByEarliest []string
	GraphRank  int
}

// Eval 是一次完整评估的结果。
type Eval struct {
	Trials    int
	Rounds    int
	Gap       time.Duration
	Alerts    int
	Incidents int
	Methods   []MethodResult
	Sample    *RoundDetail
}

// Run 跑多轮随机故障，比较三种排序方法的命中率。
func Run(trials int, seed int64, gap time.Duration, window time.Duration) *Eval {
	topo := BuildClos(2, 4, 2)
	e := &Eval{Trials: trials, Gap: gap}

	methods := []*MethodResult{
		{Name: "按告警条数排序"},
		{Name: "按最早时间排序"},
		{Name: "图打分（本 lab）"},
	}

	for i := 0; i < trials; i++ {
		rng := rand.New(rand.NewSource(seed + int64(i)*7919))
		alerts, truth := Inject(rng, topo, window)
		incs := Converge(topo, alerts, gap)

		e.Alerts += len(alerts)
		e.Incidents += len(incs)

		// 找到包含真值的 incident：真实系统里这一步是「按时间顺序逐个处置」，
		// 这里为了评估，直接定位到真值所在的 incident
		var main Inc
		found := false
		for _, inc := range incs {
			for _, ent := range inc.Entities {
				if ent == truth.Entity {
					main = Incident(inc)
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			continue
		}
		e.Rounds++

		cands := RankGraph(topo, main)
		graphRank := rankOf(candidatesToNames(cands), truth.Entity)
		countRank := rankOf(RankByCount(main), truth.Entity)
		earliestRank := rankOf(RankByEarliest(main), truth.Entity)

		methods[0].add(countRank)
		methods[1].add(earliestRank)
		methods[2].add(graphRank)

		if i == 0 {
			e.Sample = &RoundDetail{
				Truth:      truth,
				Alerts:     alerts,
				Incidents:  incs,
				MainInc:    main,
				Graph:      cands,
				ByCount:    RankByCount(main),
				ByEarliest: RankByEarliest(main),
				GraphRank:  graphRank,
			}
		}
	}

	for _, m := range methods {
		m.finish()
	}
	e.Methods = []MethodResult{*methods[0], *methods[1], *methods[2]}
	return e
}

// Inc 是 Incident 的别名，避免在循环里写长类型名。
type Inc = Incident

// Report 打印命中率对比，外加一轮明细。
func Report(w io.Writer, e *Eval) {
	fmt.Fprintf(w, "== 10-rca · 图打分 vs 朴素基线 ==\n")
	fmt.Fprintf(w, "拓扑 2 spine × 4 leaf × 2 host = 14 节点 · %d 轮随机故障 · 收敛窗口 %s\n", e.Trials, e.Gap)
	if e.Rounds != e.Trials {
		fmt.Fprintf(w, "（%d 轮有效，%d 轮真值未落在任何 incident 里，已跳过）\n",
			e.Rounds, e.Trials-e.Rounds)
	}
	fmt.Fprintln(w)

	fmt.Fprintf(w, "%-20s %10s %10s %12s\n", "方法", "Top-1", "Top-3", "平均排名")
	fmt.Println(strings.Repeat("-", 56))
	for _, m := range e.Methods {
		fmt.Fprintf(w, "%-20s %9.0f%% %9.0f%% %11.2f\n", m.Name, m.Top1*100, m.Top3*100, m.AvgRank)
	}

	if e.Incidents > 0 {
		fmt.Fprintf(w, "\n告警收敛：%d 条告警 → %d 个 incident，压缩比 %.1f×\n",
			e.Alerts, e.Incidents, float64(e.Alerts)/float64(e.Incidents))
	}

	if e.Sample != nil {
		printSample(w, e.Sample)
	}
}

func printSample(w io.Writer, s *RoundDetail) {
	fmt.Fprintf(w, "\n一轮明细（第 1 轮）\n")
	fmt.Fprintf(w, "真值：%s\n", s.Truth.Describe())
	fmt.Fprintf(w, "告警 %d 条 → %d 个 incident；取包含真值的那个，含 %d 个实体\n",
		len(s.Alerts), len(s.Incidents), len(s.MainInc.Entities))

	fmt.Fprintf(w, "\n  告警流（前 8 条）\n")
	for i, a := range s.Alerts {
		if i >= 8 {
			fmt.Fprintf(w, "  ...（共 %d 条）\n", len(s.Alerts))
			break
		}
		mark := " "
		if a.Entity == s.Truth.Entity {
			mark = "*"
		}
		fmt.Fprintf(w, "  %s %7dms  %-8s %-24s %s\n",
			mark, a.TS.Milliseconds(), a.Entity, a.Kind, a.Sev)
	}

	fmt.Fprintf(w, "\n  候选根因（图打分）\n")
	fmt.Fprintf(w, "  %-3s %-8s %7s %9s %10s %7s  %s\n", "#", "实体", "score", "coverage", "precision", "时间", "说明")
	for i, c := range s.Graph {
		if i >= 5 {
			break
		}
		note := ""
		if c.Entity == s.Truth.Entity {
			note = "← 命中真值"
		}
		fmt.Fprintf(w, "  %-3d %-8s %7.3f %8.0f%% %9.0f%% %7.2f  %s\n",
			i+1, c.Entity, c.Score, c.Coverage*100, c.Precision*100, c.TimePrior, note)
	}

	fmt.Fprintf(w, "\n  三个方法的 Top-3\n")
	fmt.Fprintf(w, "  %-20s %s\n", "图打分", strings.Join(topN(candidatesToNames(s.Graph), 3), " > "))
	fmt.Fprintf(w, "  %-20s %s\n", "按告警条数", strings.Join(topN(s.ByCount, 3), " > "))
	fmt.Fprintf(w, "  %-20s %s\n", "按最早时间", strings.Join(topN(s.ByEarliest, 3), " > "))

	fmt.Fprintf(w, "\n  读法\n")
	fmt.Fprintf(w, "  * 标出的是真值。图打分把 %s 排在第 %d 名。\n",
		s.Truth.Entity, s.GraphRank)
	fmt.Fprintf(w, "  coverage 高说明它解释得了多数告警，precision 高说明它没有靠「什么都覆盖」蒙分。\n")
	fmt.Fprintf(w, "  spine 覆盖全部节点但 precision 低，所以不会被误判成根因 —— 这正是图打分的价值。\n")
}

func topN(list []string, n int) []string {
	if len(list) <= n {
		return list
	}
	return list[:n]
}

// Topology 返回拓扑摘要，供报告头部使用。
func Topology() string {
	topo := BuildClos(2, 4, 2)
	var parts []string
	for _, tier := range []Tier{TierSpine, TierLeaf, TierHost} {
		var names []string
		for _, n := range topo.Nodes {
			if n.Tier == tier {
				names = append(names, n.Name)
			}
		}
		sort.Strings(names)
		parts = append(parts, fmt.Sprintf("%s: %s", tier, strings.Join(names, ",")))
	}
	return strings.Join(parts, "\n  ")
}
