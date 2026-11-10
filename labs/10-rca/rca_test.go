package main

import (
	"math/rand"
	"testing"
	"time"
)

func TestBuildClosDownsets(t *testing.T) {
	topo := BuildClos(2, 4, 2)

	if got := len(topo.Down["spine0"]); got != 1+4+8 {
		t.Errorf("spine0 下游 %d 个，期望 13 个（自身 + 4 leaf + 8 host）", got)
	}
	want := map[string]bool{"leaf1": true, "host2": true, "host3": true}
	for _, n := range topo.Down["leaf1"] {
		if !want[n] {
			t.Errorf("leaf1 的下游里出现了 %s", n)
		}
		delete(want, n)
	}
	if len(want) != 0 {
		t.Errorf("leaf1 的下游缺 %v", want)
	}
	if got := topo.Down["host2"]; len(got) != 1 || got[0] != "host2" {
		t.Errorf("host 的下游应当只有自身，实际 %v", got)
	}
	if !topo.IsAdjacent("spine0", "leaf0") {
		t.Error("spine0 与 leaf0 应当相邻")
	}
	if topo.IsAdjacent("host0", "host1") {
		t.Error("同 leaf 下的两台 host 之间没有直连边")
	}
}

func TestConvergeFoldsSameEntity(t *testing.T) {
	topo := BuildClos(2, 4, 2)
	alerts := []Alert{
		{TS: 0, Entity: "leaf0", Kind: "pfc-pause-high", Sev: "critical"},
		{TS: 10 * time.Millisecond, Entity: "leaf0", Kind: "queue-watermark-high", Sev: "critical"},
	}
	incs := Converge(topo, alerts, 150*time.Millisecond)
	if len(incs) != 1 {
		t.Fatalf("同实体告警应收敛成 1 个 incident，实际 %d 个", len(incs))
	}
	if len(incs[0].Alerts) != 2 {
		t.Errorf("incident 内应保留 2 条告警，实际 %d", len(incs[0].Alerts))
	}
}

func TestConvergeMergesAdjacentWithinGap(t *testing.T) {
	topo := BuildClos(2, 4, 2)
	alerts := []Alert{
		{TS: 0, Entity: "leaf0", Sev: "critical"},
		{TS: 50 * time.Millisecond, Entity: "host0", Sev: "warning"},
	}
	if got := len(Converge(topo, alerts, 150*time.Millisecond)); got != 1 {
		t.Errorf("相邻且在窗口内，应合并成 1 个 incident，实际 %d 个", got)
	}
	// 超过窗口就不该合并
	if got := len(Converge(topo, alerts, 20*time.Millisecond)); got != 2 {
		t.Errorf("超出时间窗，应保留 2 个 incident，实际 %d 个", got)
	}
	// 不相邻的实体即便时间接近也不合并
	far := []Alert{
		{TS: 0, Entity: "host0", Sev: "critical"},
		{TS: 10 * time.Millisecond, Entity: "host5", Sev: "critical"},
	}
	if got := len(Converge(topo, far, 500*time.Millisecond)); got != 2 {
		t.Errorf("不相邻的实体不该合并，实际 %d 个 incident", got)
	}
}

// 顶层节点的覆盖面最大，但 precision 必须把它压下去。
func TestPrecisionPunishesTopLevel(t *testing.T) {
	topo := BuildClos(2, 4, 2)
	inc := Incident{
		Entities: []string{"leaf1", "host2", "host3", "spine1"},
		FirstTS:  0,
		Alerts: []Alert{
			{TS: 0, Entity: "leaf1", Sev: "critical"},
			{TS: 10 * time.Millisecond, Entity: "leaf1", Sev: "critical"},
			{TS: 60 * time.Millisecond, Entity: "host2", Sev: "warning"},
			{TS: 70 * time.Millisecond, Entity: "host3", Sev: "warning"},
			{TS: 5 * time.Millisecond, Entity: "spine1", Sev: "warning"},
		},
	}
	cands := RankGraph(topo, inc)
	if len(cands) == 0 {
		t.Fatal("没有产生候选")
	}
	if cands[0].Entity != "leaf1" {
		t.Errorf("第一名是 %s，期望 leaf1（spine 覆盖更广但 precision 低）", cands[0].Entity)
	}
	// 另一台 spine 覆盖不到兄弟 spine 的告警，coverage 不该是满分
	for _, c := range cands {
		if c.Entity == "spine0" && c.Coverage >= 1 {
			t.Errorf("spine0 的下游不含 spine1，coverage 不该是 %.2f", c.Coverage)
		}
	}
	var spine1 Candidate
	for _, c := range cands {
		if c.Entity == "spine1" {
			spine1 = c
		}
	}
	if spine1.Coverage < 0.9 {
		t.Errorf("spine1 应覆盖全部告警，实际 %.2f", spine1.Coverage)
	}
	if spine1.Precision > 0.5 {
		t.Errorf("spine1 的 precision 应被分母拉低，实际 %.2f", spine1.Precision)
	}
}

func TestInjectProducesOrderedAlerts(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	topo := BuildClos(2, 4, 2)
	alerts, truth := Inject(rng, topo, 10*time.Second)

	if len(alerts) < 5 {
		t.Errorf("告警只有 %d 条，太少了", len(alerts))
	}
	for i := 1; i < len(alerts); i++ {
		if alerts[i].TS < alerts[i-1].TS {
			t.Fatalf("告警流没按时间排序：第 %d 条 %v 早于前一条 %v", i, alerts[i].TS, alerts[i-1].TS)
		}
	}
	// 根因必须有告警，否则无法评估
	found := false
	for _, a := range alerts {
		if a.Entity == truth.Entity && a.Sev == "critical" {
			found = true
		}
	}
	if !found {
		t.Errorf("根因 %s 没有 critical 告警", truth.Entity)
	}
}

func TestCriticalWeighsMoreThanWarning(t *testing.T) {
	critical := Alert{Sev: "critical"}
	warning := Alert{Sev: "warning"}
	if critical.Weight() <= warning.Weight() {
		t.Errorf("critical 权重 %.1f 应高于 warning %.1f", critical.Weight(), warning.Weight())
	}
}

// 这是本 lab 的核心断言：图打分必须赢过两个朴素基线。
func TestGraphBeatsBaselines(t *testing.T) {
	e := Run(200, 42, 150*time.Millisecond, 10*time.Second)
	if len(e.Methods) != 3 {
		t.Fatalf("方法数 %d，期望 3", len(e.Methods))
	}
	graph := e.Methods[2]
	count := e.Methods[0]
	earliest := e.Methods[1]

	if graph.Top1 <= count.Top1 {
		t.Errorf("图打分 Top-1 %.0f%% 没有超过按告警条数 %.0f%%", graph.Top1*100, count.Top1*100)
	}
	if graph.Top1 < earliest.Top1 {
		t.Errorf("图打分 Top-1 %.0f%% 低于按最早时间 %.0f%%", graph.Top1*100, earliest.Top1*100)
	}
	if graph.Top3 < 0.95 {
		t.Errorf("图打分 Top-3 只有 %.0f%%，太低", graph.Top3*100)
	}
	if graph.AvgRank > 1.3 {
		t.Errorf("图打分平均排名 %.2f，偏高", graph.AvgRank)
	}
	// 收敛必须真的压缩了数据
	if e.Incidents*3 > e.Alerts {
		t.Errorf("压缩比只有 %.1f×，收敛没起作用", float64(e.Alerts)/float64(e.Incidents))
	}
}

func TestEvalDeterministic(t *testing.T) {
	a := Run(30, 7, 150*time.Millisecond, 10*time.Second)
	b := Run(30, 7, 150*time.Millisecond, 10*time.Second)
	if a.Methods[2].Top1 != b.Methods[2].Top1 || a.Alerts != b.Alerts {
		t.Errorf("同种子两次评估结果不一致：%v vs %v", a.Methods[2].Top1, b.Methods[2].Top1)
	}
}
