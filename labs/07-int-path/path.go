package main

import "fmt"

// Scenario 是四种典型路径形态。归因判读靠它们的组合，不是靠单一阈值。
type Scenario string

const (
	// ScHealthy 全程健康：低时延、低队列、中等利用率。
	ScHealthy Scenario = "healthy"
	// ScEgressCongested 某一跳出口被打满：利用率高 + 队列高。
	ScEgressCongested Scenario = "egress"
	// ScUpstream 某一跳队列高但出口利用率不高：多股流在抢，问题在上游分布。
	ScUpstream Scenario = "upstream"
	// ScForwarding 某一跳时延高但队列与利用率都低：查表、ACL、微码这些转发面开销。
	ScForwarding Scenario = "forwarding"
)

// Scenarios 供 flag 校验与 usage 输出。
var Scenarios = []Scenario{ScHealthy, ScEgressCongested, ScUpstream, ScForwarding}

// ParseScenario 校验输入。
func ParseScenario(s string) (Scenario, error) {
	for _, sc := range Scenarios {
		if Scenario(s) == sc {
			return sc, nil
		}
	}
	return "", fmt.Errorf("未知场景 %q，可选 %v", s, Scenarios)
}

// Path 是一次采样报文经过的完整跳序列。
type Path struct {
	Scenario Scenario
	Hops     []HopMeta
	// 受害流标识，用来把 INT 记录与训练任务对上
	FlowID uint32
}

// BuildPath 造一条 5 跳的路径。
//
// 4 台交换机 + 1 台出口设备，端口编号用 map 里的约定值，
// 不出现任何真实设备配置。
func BuildPath(sc Scenario) Path {
	base := []HopMeta{
		{SwitchID: 0x11, Ingress: 1, Egress: 8, HopLatNs: 800, QueueKB: 12, TxUtilBP: 4200},
		{SwitchID: 0x2a, Ingress: 3, Egress: 11, HopLatNs: 900, QueueKB: 18, TxUtilBP: 5100},
		{SwitchID: 0x3f, Ingress: 7, Egress: 2, HopLatNs: 850, QueueKB: 20, TxUtilBP: 4800},
		{SwitchID: 0x4c, Ingress: 5, Egress: 13, HopLatNs: 950, QueueKB: 22, TxUtilBP: 5500},
		{SwitchID: 0x5d, Ingress: 9, Egress: 4, HopLatNs: 780, QueueKB: 15, TxUtilBP: 4600},
	}
	hops := make([]HopMeta, len(base))
	copy(hops, base)

	switch sc {
	case ScEgressCongested:
		// 第 3 跳出口打满：利用率 97%，队列堆到 780 KB
		hops[2].TxUtilBP = 9700
		hops[2].QueueKB = 780
		hops[2].HopLatNs = 42000
	case ScUpstream:
		// 第 3 跳队列高但出口只有 55%：多股流在抢同一个出口
		hops[2].TxUtilBP = 5500
		hops[2].QueueKB = 640
		hops[2].HopLatNs = 28000
	case ScForwarding:
		// 第 3 跳时延 40 µs，但队列与利用率都低：转发面慢
		hops[2].TxUtilBP = 4000
		hops[2].QueueKB = 16
		hops[2].HopLatNs = 40000
	}

	return Path{Scenario: sc, Hops: hops, FlowID: 0x7a31}
}

// TotalLatNs 是逐跳时延之和，单位纳秒。
func (p Path) TotalLatNs() uint32 {
	var sum uint32
	for _, h := range p.Hops {
		sum += h.HopLatNs
	}
	return sum
}

// Payload 返回这条路径的线格式载荷。
func (p Path) Payload() []byte { return Encode(p.Hops) }
