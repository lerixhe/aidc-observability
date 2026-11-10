package main

import (
	"bytes"
	"math"
	"testing"
)

// 这套测试守的是判读逻辑，不是网络行为：
// 四类故障的样本必须被分到四类，且判据（丢包/周期/pause/时延）能各自起作用。

func TestEachScenarioClassifiesAsItself(t *testing.T) {
	th := DefaultThresholds()
	for _, s := range Scenarios() {
		s.Seed = 42
		rows := Simulate(s)
		v := Classify(rows, th)
		if v.Kind != s.Kind {
			t.Errorf("场景 %s 被判成 %s（期望 %s）\n依据：%v", s.Name, v.Kind, s.Kind, v.Evidence)
		}
	}
}

// 换种子只该影响抖动幅度，不该翻转结论。翻不过来就说明判据贴着阈值走，太脆。
func TestVerdictStableAcrossSeeds(t *testing.T) {
	th := DefaultThresholds()
	for _, s := range Scenarios() {
		for _, seed := range []int64{1, 7, 42, 2024} {
			s.Seed = seed
			v := Classify(Simulate(s), th)
			if v.Kind != s.Kind {
				t.Errorf("场景 %s 在 seed=%d 被判成 %s，判据不稳", s.Name, seed, v.Kind)
			}
		}
	}
}

// incast 的指纹是周期性：尖刺间隔应接近场景设定的 4 秒。
func TestIncastIsPeriodic(t *testing.T) {
	s, _ := FindScenario("incast")
	s.Seed = 42
	f := Extract(Simulate(s), DefaultThresholds())
	if !f.Periodic {
		t.Fatalf("incast 没被判成周期性，特征：%+v", f)
	}
	if math.Abs(f.Period-4) > 1.5 {
		t.Errorf("尖刺间隔 %.2f 秒，与设定的 4 秒差太多", f.Period)
	}
}

// 风暴与队头阻塞的分界在时延倍数与 pause 强度上，这条测试把分界钉住。
func TestStormVersusHOLThreshold(t *testing.T) {
	th := DefaultThresholds()
	if th.PauseSevere >= 20 {
		t.Errorf("pause 严重线 %.1f%% 太高，风暴与 HoL 会混在一起", th.PauseSevere)
	}

	storm := Classify(simulateNamed(t, "storm"), th)
	hol := Classify(simulateNamed(t, "hol"), th)
	if storm.Features.PauseMax <= hol.Features.PauseMax {
		t.Errorf("风暴的 pause 峰值 %.1f 不该低于 HoL 的 %.1f",
			storm.Features.PauseMax, hol.Features.PauseMax)
	}
	if storm.Features.LatRatio <= hol.Features.LatRatio {
		t.Errorf("风暴的时延倍数 %.1f 不该低于 HoL 的 %.1f",
			storm.Features.LatRatio, hol.Features.LatRatio)
	}
}

// 死锁的指纹是「超时 + 不丢包」，两个条件缺一不可。
func TestDeadlockNeedsTimeoutWithoutLoss(t *testing.T) {
	v := Classify(simulateNamed(t, "deadlock"), DefaultThresholds())
	if v.Features.TimeoutFrac < 0.15 {
		t.Errorf("死锁样本的超时占比只有 %.0f%%，样本形态不对", v.Features.TimeoutFrac*100)
	}
	if v.Features.LossMax > 0.05 {
		t.Errorf("死锁样本不该有丢包，实际 %.2f%%", v.Features.LossMax)
	}
}

// 关掉 PFC 的等价形态：有丢包但没周期性，应判到 lossy 而不是 incast。
func TestSteadyLossIsNotIncast(t *testing.T) {
	s, _ := FindScenario("incast")
	s.Seed = 42
	for i := range s.Ports {
		s.Ports[i].Period = 0
		s.Ports[i].Duty = 1
	}
	v := Classify(Simulate(s), DefaultThresholds())
	if v.Kind == "incast" {
		t.Errorf("持续丢包被误判成 incast，周期性判据失效：%v", v.Evidence)
	}
}

// CSV 往返：写出去的样本能被读回来，数值不漂。
func TestSampleCSVRoundTrip(t *testing.T) {
	s, _ := FindScenario("hol")
	s.Seed = 42
	rows := Simulate(s)

	var buf bytes.Buffer
	if err := WriteSampleCSV(&buf, rows); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if _, err := SaveSample(dir, s, rows); err != nil {
		t.Fatal(err)
	}
	back, err := LoadSample(dir + "/model-hol.csv")
	if err != nil {
		t.Fatal(err)
	}
	if len(back) != len(rows) {
		t.Fatalf("往返后行数 %d，原 %d", len(back), len(rows))
	}
	for i := range rows {
		if back[i].Port != rows[i].Port || math.Abs(back[i].PausePct-rows[i].PausePct) > 0.01 {
			t.Fatalf("第 %d 行不一致：%+v vs %+v", i, back[i], rows[i])
		}
	}
}

// 注入计划必须引用拓扑里真实存在的节点，容器名拼错是最常见的浪费时间的坑。
func TestPlanTargetsExistInTopology(t *testing.T) {
	nodes := map[string]bool{
		"h1": true, "h2": true, "h3": true, "h4": true,
		"leaf1": true, "leaf2": true, "spine1": true,
	}
	for _, s := range Scenarios() {
		if len(s.Injection) == 0 {
			t.Errorf("场景 %s 没有注入步骤", s.Name)
		}
		for _, st := range s.Injection {
			if !nodes[st.Node] {
				t.Errorf("场景 %s 的注入步骤指向不存在的节点 %q", s.Name, st.Node)
			}
			if st.Cmd == "" {
				t.Errorf("场景 %s 在节点 %s 上有空命令", s.Name, st.Node)
			}
		}
	}
}

func simulateNamed(t *testing.T, name string) []Row {
	t.Helper()
	s, ok := FindScenario(name)
	if !ok {
		t.Fatalf("没有场景 %s", name)
	}
	s.Seed = 42
	return Simulate(s)
}
