package main

import (
	"math"
	"sort"
	"time"
)

// Incident 是收敛后的一个事件：一组相关实体 + 该事件的第一条告警。
type Incident struct {
	Entities []string
	FirstTS  time.Duration
	Alerts   []Alert
}

// Converge 把告警压成 incident。
//
// 规则只有两条：同一实体的告警折叠；拓扑相邻且时间靠近的实体合并。
// 这就是课程里「同设备同时间窗折叠 + 按拓扑归并」的最简实现。
func Converge(topo *Topo, alerts []Alert, gap time.Duration) []Incident {
	if len(alerts) == 0 {
		return nil
	}

	first := map[string]time.Duration{}
	byEntity := map[string][]Alert{}
	for _, a := range alerts {
		if t, ok := first[a.Entity]; !ok || a.TS < t {
			first[a.Entity] = a.TS
		}
		byEntity[a.Entity] = append(byEntity[a.Entity], a)
	}

	parent := map[string]string{}
	var find func(string) string
	find = func(x string) string {
		if parent[x] == "" {
			parent[x] = x
		}
		if parent[x] != x {
			parent[x] = find(parent[x])
		}
		return parent[x]
	}
	union := func(a, b string) {
		ra, rb := find(a), find(b)
		if ra != rb {
			parent[ra] = rb
		}
	}

	entities := make([]string, 0, len(first))
	for e := range first {
		entities = append(entities, e)
	}
	sort.Strings(entities)

	// 相邻 + 时间相近 → 同源
	for i := 0; i < len(entities); i++ {
		for j := i + 1; j < len(entities); j++ {
			a, b := entities[i], entities[j]
			if !topo.IsAdjacent(a, b) {
				continue
			}
			if absDuration(first[a]-first[b]) <= gap {
				union(a, b)
			}
		}
	}

	groups := map[string][]string{}
	for _, e := range entities {
		r := find(e)
		groups[r] = append(groups[r], e)
	}

	roots := make([]string, 0, len(groups))
	for r := range groups {
		roots = append(roots, r)
	}
	sort.Strings(roots)

	out := make([]Incident, 0, len(roots))
	for _, r := range roots {
		members := groups[r]
		inc := Incident{Entities: members}
		inc.FirstTS = first[members[0]]
		for _, e := range members {
			if first[e] < inc.FirstTS {
				inc.FirstTS = first[e]
			}
			inc.Alerts = append(inc.Alerts, byEntity[e]...)
		}
		sort.SliceStable(inc.Alerts, func(i, j int) bool { return inc.Alerts[i].TS < inc.Alerts[j].TS })
		out = append(out, inc)
	}
	// 时间最早的 incident 排前面：真实排障也是先处理第一个爆的
	sort.SliceStable(out, func(i, j int) bool { return out[i].FirstTS < out[j].FirstTS })
	return out
}

// Candidate 是一个根因候选及其打分拆解。拆解比总分更有用：
// 值班时你要能解释「为什么它是第一名」。
type Candidate struct {
	Entity    string
	Score     float64
	Coverage  float64 // 按严重度加权后，它能解释的告警占比
	Precision float64 // 它的下游里有多少真的报了警
	TimePrior float64 // 它是不是早于下游就报了警
	Explained float64 // 加权解释量
}

// RankGraph 用「共同上游祖先 + 传播路径评分」给候选排序。
//
// 打分 = F1(coverage, precision) × (0.4 + 0.6 × 时间先验)
//
//	coverage  按严重度加权：critical 比 warning 更能指认根因。
//	          不加权的话，一台设备的海量 warning 会把根因那两条 critical 淹掉。
//	precision 越大 → 它没有把无关分支也算进来（惩罚「什么都覆盖」的顶层节点）
//	时间先验  越大 → 它的告警比下游更早，符合「故障向下流」
func RankGraph(topo *Topo, inc Incident) []Candidate {
	if len(inc.Entities) == 0 {
		return nil
	}
	firstOf := map[string]time.Duration{}
	for _, a := range inc.Alerts {
		if t, ok := firstOf[a.Entity]; !ok || a.TS < t {
			firstOf[a.Entity] = a.TS
		}
	}
	span := incSpan(inc)
	if span <= 0 {
		span = 100 * time.Millisecond
	}

	totalWeight := 0.0
	entityWeight := map[string]float64{}
	for _, a := range inc.Alerts {
		totalWeight += a.Weight()
		entityWeight[a.Entity] += a.Weight()
	}

	out := make([]Candidate, 0, len(topo.Nodes))
	for _, name := range topo.Names() {
		cand := Candidate{Entity: name}
		for _, e := range inc.Entities {
			if topo.Contains(name, e) {
				cand.Explained += entityWeight[e]
			}
		}
		if cand.Explained == 0 {
			continue
		}
		cand.Coverage = cand.Explained / totalWeight
		// precision 按实体计数算，不加权：它问的是「下游有没有无辜分支」，
		// 用加权量会算出大于 1 的无意义值
		hit := 0
		for _, e := range inc.Entities {
			if topo.Contains(name, e) && entityWeight[e] > 0 {
				hit++
			}
		}
		cand.Precision = float64(hit) / float64(len(topo.Down[name]))

		var f1 float64
		if cand.Coverage+cand.Precision > 0 {
			f1 = 2 * cand.Coverage * cand.Precision / (cand.Coverage + cand.Precision)
		}

		// 没有自己的告警 → 没给出早期信号，只拿半份时间分
		if t, ok := firstOf[name]; ok {
			lag := t - inc.FirstTS
			cand.TimePrior = clamp01(1 - float64(lag)/float64(span))
		} else {
			cand.TimePrior = 0.5
		}

		cand.Score = f1 * (0.4 + 0.6*cand.TimePrior)
		out = append(out, cand)
	}

	sort.SliceStable(out, func(i, j int) bool {
		if math.Abs(out[i].Score-out[j].Score) > 1e-9 {
			return out[i].Score > out[j].Score
		}
		// 同分时偏向更靠下的节点：故障向下流，越靠下越具体
		return topoTierRank(topo, out[i].Entity) > topoTierRank(topo, out[j].Entity)
	})
	return out
}

// RankByCount 是最朴素的基线：谁的告警最多谁的嫌疑最大。
func RankByCount(inc Incident) []string {
	count := map[string]int{}
	var order []string
	for _, a := range inc.Alerts {
		if count[a.Entity] == 0 {
			order = append(order, a.Entity)
		}
		count[a.Entity]++
	}
	sort.SliceStable(order, func(i, j int) bool { return count[order[i]] > count[order[j]] })
	return order
}

// RankByEarliest 是第二个基线：谁最先报警谁是根因。
// 这条规则在单向传播里有效，但会被「上游反向告警」带偏。
func RankByEarliest(inc Incident) []string {
	first := map[string]time.Duration{}
	var order []string
	for _, a := range inc.Alerts {
		if t, ok := first[a.Entity]; !ok {
			first[a.Entity] = a.TS
			order = append(order, a.Entity)
		} else if a.TS < t {
			first[a.Entity] = a.TS
		}
	}
	sort.SliceStable(order, func(i, j int) bool { return first[order[i]] < first[order[j]] })
	return order
}

// rankOf 返回实体在有序列表里的名次（1-based），找不到返回 len+1。
func rankOf(list []string, entity string) int {
	for i, e := range list {
		if e == entity {
			return i + 1
		}
	}
	return len(list) + 1
}

func candidatesToNames(cands []Candidate) []string {
	out := make([]string, len(cands))
	for i, c := range cands {
		out[i] = c.Entity
	}
	return out
}

func incSpan(inc Incident) time.Duration {
	if len(inc.Alerts) == 0 {
		return 0
	}
	lo, hi := inc.Alerts[0].TS, inc.Alerts[0].TS
	for _, a := range inc.Alerts {
		if a.TS < lo {
			lo = a.TS
		}
		if a.TS > hi {
			hi = a.TS
		}
	}
	return hi - lo
}

func topoTierRank(topo *Topo, name string) int {
	return int(topo.TierOf(name))
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}
