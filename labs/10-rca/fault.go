package main

import (
	"fmt"
	"math/rand"
	"sort"
	"time"
)

// Alert 是一条告警。真实系统里字段会多得多，这里只留 RCA 用得上的几个。
type Alert struct {
	TS     time.Duration
	Entity string
	Kind   string
	Sev    string
}

// Weight 是告警的信息量权重。
//
// 分级不是装饰：critical 通常只有根因才发得出来，warning 谁都能发。
// RCA 拿同一把尺子量所有告警，就会被海量 warning 淹没。
func (a Alert) Weight() float64 {
	switch a.Sev {
	case "critical":
		return 3
	case "warning":
		return 1
	}
	return 0.5
}

// Truth 是这一轮注入的故障真值，评估时用来判命中。
type Truth struct {
	Entity string
	Type   string
	T0     time.Duration
}

// Inject 生成一轮「一个故障 + 一堆派生告警 + 噪声」的告警流。
//
// 传播假设与课程一致：故障沿拓扑向下流。
// 根因自身最先告警，下游按层级延迟递增，上游偶尔回一条反向告警当噪声。
func Inject(rng *rand.Rand, topo *Topo, window time.Duration) ([]Alert, Truth) {
	root := topo.Nodes[rng.Intn(len(topo.Nodes))]
	t0 := time.Duration(rng.Int63n(int64(window / 2)))
	ftype := faultTypeFor(root.Tier, rng)

	truth := Truth{Entity: root.Name, Type: ftype, T0: t0}
	var alerts []Alert

	// ① 根因自身：两条 critical。
	// 采集链路不是零延迟，加 0~60 ms 抖动 —— 真实环境里下游告警偶尔会比根因先到，
	// 「谁先报谁是根因」这条规则就是这么被打破的。
	for i, kind := range rootAlarmKinds(root.Tier) {
		alerts = append(alerts, Alert{
			TS:     t0 + time.Duration(i)*8*time.Millisecond + time.Duration(rng.Intn(60))*time.Millisecond,
			Entity: root.Name,
			Kind:   kind,
			Sev:    "critical",
		})
	}

	// ② 下游：按层级深度加延迟，每层叠一点抖动
	for _, name := range topo.Down[root.Name] {
		if name == root.Name {
			continue
		}
		depth := topo.TierOf(name) - topo.TierOf(root.Name)
		n := 3 + rng.Intn(6) // 每个下游节点 3~8 条：路径探测、队列、任务侧各报各的
		for i := 0; i < n; i++ {
			delay := time.Duration(depth)*60*time.Millisecond +
				time.Duration(rng.Intn(90))*time.Millisecond +
				time.Duration(i)*15*time.Millisecond
			alerts = append(alerts, Alert{
				TS:     t0 + delay,
				Entity: name,
				Kind:   downstreamAlarmKind(topo.TierOf(name), rng),
				Sev:    "warning",
			})
		}
	}

	// ③ 上游回一条反向告警：真实网络里下游异常会被上游看到，方向容易误导
	if up := topo.Adj[root.Name]; len(up) > 0 {
		peer := up[rng.Intn(len(up))]
		alerts = append(alerts, Alert{
			TS:     t0 + time.Duration(rng.Intn(100))*time.Millisecond,
			Entity: peer,
			Kind:   "neighbor-down",
			Sev:    "warning",
		})
	}

	// ④ 噪声告警：与本次故障无关。
	// 集中在少数几个实体上，因为真实噪声也是「某个设备在反复抖」，
	// 不是均匀撒胡椒面 —— 这决定了收敛能压掉多少。
	for i := 0; i < 2+rng.Intn(3); i++ {
		name := topo.Nodes[rng.Intn(len(topo.Nodes))].Name
		for j := 0; j < 1+rng.Intn(3); j++ {
			alerts = append(alerts, Alert{
				TS:     time.Duration(rng.Int63n(int64(window))),
				Entity: name,
				Kind:   downstreamAlarmKind(topo.TierOf(name), rng),
				Sev:    "warning",
			})
		}
	}

	// 真实告警流是乱序到达的
	sort.SliceStable(alerts, func(i, j int) bool { return alerts[i].TS < alerts[j].TS })
	return alerts, truth
}

func faultTypeFor(tier Tier, rng *rand.Rand) string {
	switch tier {
	case TierSpine:
		return []string{"pfc-storm", "link-degraded", "congestion-config"}[rng.Intn(3)]
	case TierLeaf:
		return []string{"pfc-storm", "queue-misconfig", "optics-degrade"}[rng.Intn(3)]
	default:
		return []string{"nic-fault", "app-stall"}[rng.Intn(2)]
	}
}

func rootAlarmKinds(tier Tier) []string {
	switch tier {
	case TierHost:
		return []string{"nic-retransmit-high", "task-slow"}
	case TierLeaf:
		return []string{"pfc-pause-high", "queue-watermark-high"}
	default:
		return []string{"pfc-pause-high", "queue-watermark-high"}
	}
}

func downstreamAlarmKind(tier Tier, rng *rand.Rand) string {
	switch tier {
	case TierHost:
		return []string{"task-slow", "nccl-timeout", "path-latency-high"}[rng.Intn(3)]
	case TierLeaf:
		return []string{"path-latency-high", "traffic-shift", "ecn-marked-high"}[rng.Intn(3)]
	default:
		return []string{"path-latency-high", "bgp-jitter"}[rng.Intn(2)]
	}
}

// Describe 给报告用的一句话描述。
func (t Truth) Describe() string {
	return fmt.Sprintf("%s（%s 层）· %s · t0=%dms", t.Entity, tierName(t.Entity), t.Type, t.T0.Milliseconds())
}

func tierName(entity string) string {
	switch {
	case len(entity) >= 5 && entity[:5] == "spine":
		return "spine"
	case len(entity) >= 4 && entity[:4] == "leaf":
		return "leaf"
	}
	return "host"
}
