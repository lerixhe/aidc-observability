package main

import (
	"math"
	"strings"
	"testing"
)

// 这些测试守的是三件事：
//  1. 散列是确定性的，同样的输入永远同一个结果（否则 expected 没法锁）
//  2. 三层口径忽略端口，五元组口径用端口 —— 这是整个 lab 的教学点
//  3. 流数越多越均匀，否则「大数定律」那句结论就是错的

func TestHashDeterministic(t *testing.T) {
	f := Flow{Src: 1, Dst: 2, SPort: 3, DPort: 4, Proto: 6}
	a := FibMultipathHash(f, PolicyL4)
	b := FibMultipathHash(f, PolicyL4)
	if a != b {
		t.Fatalf("同一输入两次散列结果不同：%d vs %d", a, b)
	}
}

// 三层口径只看 IP：源目固定、端口不同的一批流，必须全部落到同一条链路。
func TestL3IgnoresPorts(t *testing.T) {
	flows := Generate(WorkSameTuple, 64, 42)
	c := map[int]int{}
	for _, f := range flows {
		c[PickLink(FibMultipathHash(f, PolicyL3), 2)]++
	}
	if len(c) != 1 {
		t.Fatalf("三层口径下端口不该影响选择，实际散到了 %d 条链路：%v", len(c), c)
	}
}

// 五元组口径吃端口：同一对 IP 的连接会被摊到多条链路。
func TestL4UsesPorts(t *testing.T) {
	flows := Generate(WorkSameTuple, 64, 42)
	c := map[int]int{}
	for _, f := range flows {
		c[PickLink(FibMultipathHash(f, PolicyL4), 2)]++
	}
	if len(c) != 2 {
		t.Fatalf("五元组口径下 64 条不同端口的流应摊到 2 条链路，实际 %d 条：%v", len(c), c)
	}
	if c[0] == 0 || c[1] == 0 {
		t.Fatalf("两条链路该都分到流量：%v", c)
	}
}

// 流数上去，极化度应该下来。这条如果不成立，实验一的结论就是编的。
func TestMoreFlowsMeansLessPolarization(t *testing.T) {
	small := Simulate(Generate(WorkRandom, 8, 42), PolicyL4)
	large := Simulate(Generate(WorkRandom, 2048, 42), PolicyL4)
	if large.Worst >= small.Worst {
		t.Errorf("2048 条流的极化度 %.2f 不该高于 8 条流的 %.2f", large.Worst, small.Worst)
	}
	if large.CV > 0.15 {
		t.Errorf("2048 条流下变异系数 %.2f 偏大，散列分布可能有问题", large.CV)
	}
}

// 同质大象流在任何口径下都只能走一条路：输入没有新信息，算法无从选择。
func TestElephantFlowsNeverSpread(t *testing.T) {
	for _, p := range []HashPolicy{PolicyL3, PolicyL4} {
		res := Simulate(Generate(WorkElephant, 64, 42), p)
		if res.Worst < 1.99 {
			t.Errorf("口径 %d：同质大象流的极化度 %.2f，本该是 2.00（只用到一条上行）", p, res.Worst)
		}
	}
}

// 同 leaf 内互访不走上行，必须计入 Local 而不是混进链路统计。
func TestLocalFlowsExcluded(t *testing.T) {
	hosts := Hosts()
	flows := []Flow{
		{Src: hosts[0].IP, Dst: hosts[1].IP, SPort: 1, DPort: 2, Proto: 6, Bytes: 1},
		{Src: hosts[0].IP, Dst: hosts[6].IP, SPort: 1, DPort: 2, Proto: 6, Bytes: 1},
	}
	res := Simulate(flows, PolicyL4)
	if res.Local != 1 {
		t.Errorf("同 leaf 互访应有 1 条，实际 %d", res.Local)
	}
}

// PickLink 的边界：一条链路时永远选它，零条也不该 panic。
func TestPickLinkEdges(t *testing.T) {
	if got := PickLink(12345, 1); got != 0 {
		t.Errorf("单链路应该返回 0，实际 %d", got)
	}
	if got := PickLink(12345, 0); got != 0 {
		t.Errorf("零链路应该安全返回 0，实际 %d", got)
	}
	for i := 0; i < 100; i++ {
		if got := PickLink(uint32(i), 2); got < 0 || got > 1 {
			t.Fatalf("两链路时返回了越界下标 %d", got)
		}
	}
}

// 拓扑地址表要自洽：上行两端在同一 /30 段内，且 leaf 侧主机号是 1。
func TestUplinkAddressing(t *testing.T) {
	seen := map[uint32]bool{}
	for _, u := range Uplinks() {
		if u.Local>>2 != u.Remote>>2 {
			t.Errorf("%s→%s 两端不在同一个 /30：%s / %s",
				u.Leaf, u.Spine, ipString(u.Local), ipString(u.Remote))
		}
		if u.Local&3 != 1 || u.Remote&3 != 2 {
			t.Errorf("%s→%s 主机号不是 1/2：%d/%d", u.Leaf, u.Spine, u.Local&3, u.Remote&3)
		}
		if seen[u.Local] {
			t.Errorf("地址 %s 重复分配", ipString(u.Local))
		}
		seen[u.Local] = true
	}
	if len(seen) != 8 {
		t.Errorf("应有 8 条上行，实际 %d", len(seen))
	}
}

// 极化度与 CV 的算术检查，防止后续改统计口径时算错。
func TestPolarizationMath(t *testing.T) {
	ls := LeafStat{
		Leaf:     "leaf1",
		Links:    []LinkStat{{Flows: 6, Share: 0.75}, {Flows: 2, Share: 0.25}},
		MaxShare: 0.75,
	}
	ls.Ideal = 0.5
	ls.Polar = ls.MaxShare / ls.Ideal
	if math.Abs(ls.Polar-1.5) > 1e-9 {
		t.Errorf("6:2 的极化度应为 1.5，实际 %.2f", ls.Polar)
	}
	if got := cv([]float64{0.75, 0.25}); math.Abs(got-0.5) > 1e-9 {
		t.Errorf("75/25 分布的变异系数应为 0.5，实际 %.2f", got)
	}
}

func TestMermaidOutput(t *testing.T) {
	flows := Generate(WorkRandom, 64, 42)
	res := Simulate(flows, PolicyL4)
	var b strings.Builder
	ReportMermaid(&b, res)
	out := b.String()
	if !strings.Contains(out, "graph TD") {
		t.Error("Mermaid 图缺少 graph TD 头部")
	}
	if !strings.Contains(out, "spine1") || !strings.Contains(out, "leaf1") {
		t.Error("Mermaid 图缺少节点定义")
	}
	if !strings.Contains(out, "linkStyle") {
		t.Error("Mermaid 图缺少着色样式")
	}
}
