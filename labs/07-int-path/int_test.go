package main

import (
	"math"
	"testing"
)

func TestEncodeDecodeRoundTrip(t *testing.T) {
	p := BuildPath(ScEgressCongested)
	got, err := Decode(p.Payload())
	if err != nil {
		t.Fatalf("解码失败：%v", err)
	}
	if len(got) != len(p.Hops) {
		t.Fatalf("跳数不一致：want %d, got %d", len(p.Hops), len(got))
	}
	for i := range got {
		if got[i] != p.Hops[i] {
			t.Fatalf("第 %d 跳不一致：want %+v, got %+v", i+1, p.Hops[i], got[i])
		}
	}
}

func TestWireSize(t *testing.T) {
	p := BuildPath(ScHealthy)
	if got := len(p.Payload()); got != WireBytes(len(p.Hops)) {
		t.Fatalf("载荷长度 want %d, got %d", WireBytes(len(p.Hops)), got)
	}
	if WireBytes(8) != 192 {
		t.Fatalf("8 跳应为 192 字节，got %d", WireBytes(8))
	}
}

func TestDecodeRejectsTruncated(t *testing.T) {
	payload := BuildPath(ScHealthy).Payload()
	cases := [][]byte{
		payload[:HopSize+1],
		payload[:HopSize-1],
		[]byte{0x01},
	}
	for i, c := range cases {
		if _, err := Decode(c); err == nil {
			t.Fatalf("用例 %d：截断载荷应当报错", i)
		}
	}
	if _, err := Decode(nil); err != nil {
		t.Fatalf("空载荷应当返回空结果而不报错，got %v", err)
	}
}

func TestClassifyHop(t *testing.T) {
	cases := []struct {
		name string
		hop  HopMeta
		want Verdict
	}{
		{"健康跳", HopMeta{HopLatNs: 800, QueueKB: 12, TxUtilBP: 4200}, VerdictOK},
		{"出口打满加队列堆积", HopMeta{HopLatNs: 42000, QueueKB: 780, TxUtilBP: 9700}, VerdictEgress},
		{"队列高但出口闲", HopMeta{HopLatNs: 28000, QueueKB: 640, TxUtilBP: 5500}, VerdictUpstream},
		{"队列空而停留久", HopMeta{HopLatNs: 40000, QueueKB: 16, TxUtilBP: 4000}, VerdictForwarding},
		{"停留久但两个量都不典型", HopMeta{HopLatNs: 25000, QueueKB: 100, TxUtilBP: 9000}, VerdictUnclearNote},
		{"时延不高但有堆积", HopMeta{HopLatNs: 900, QueueKB: 900, TxUtilBP: 9900}, VerdictOK},
	}
	for _, c := range cases {
		got, _ := ClassifyHop(c.hop)
		if got != c.want {
			t.Errorf("%s：want %s, got %s", c.name, c.want, got)
		}
	}
}

func TestAnalyzeShares(t *testing.T) {
	p := BuildPath(ScEgressCongested)
	a := Analyze(p.FlowID, p.Hops)

	if a.TotalNs != p.TotalLatNs() {
		t.Fatalf("总时延不一致：want %d, got %d", p.TotalLatNs(), a.TotalNs)
	}
	var sum float64
	for _, h := range a.Hops {
		sum += h.SharePc
	}
	if math.Abs(sum-100) > 0.01 {
		t.Fatalf("占比之和应为 100，got %.3f", sum)
	}
	if a.WorstIndex != 2 {
		t.Fatalf("最慢跳应为第 3 跳（索引 2），got %d", a.WorstIndex)
	}
	if len(a.Suspects()) != 1 {
		t.Fatalf("egress 场景应只有一个可疑跳，got %d", len(a.Suspects()))
	}
}

func TestAnalyzeHealthyHasNoSuspect(t *testing.T) {
	p := BuildPath(ScHealthy)
	a := Analyze(p.FlowID, p.Hops)
	if got := len(a.Suspects()); got != 0 {
		t.Fatalf("健康路径不该有可疑跳，got %d", got)
	}
	if a.Headline() == "" {
		t.Fatal("结论不该为空")
	}
}

func TestCaptureProbMonotonic(t *testing.T) {
	var prev float64 = 1.1
	for _, k := range SampleRates {
		p := CaptureProb(DefaultPktRate, DefaultBurstUs, k)
		if p <= 0 || p >= 1 {
			t.Fatalf("概率应在 0 到 1 之间，1:%d got %.4f", k, p)
		}
		if p >= prev {
			t.Fatalf("采样率越粗，捕获率应越低：1:%d got %.4f，上一档 %.4f", k, p, prev)
		}
		prev = p
	}
}

func TestCaptureProbMatchesMonteCarlo(t *testing.T) {
	rows := RunSampling(DefaultPktRate, DefaultBurstUs, 7, 4000)
	for _, r := range rows {
		if math.Abs(r.Analytic-r.Measured) > 0.03 {
			t.Errorf("1:%d 解析 %.3f 与实测 %.3f 偏差过大", r.Rate, r.Analytic, r.Measured)
		}
	}
}

func TestRunSamplingDeterministic(t *testing.T) {
	a := RunSampling(DefaultPktRate, DefaultBurstUs, 42, 500)
	b := RunSampling(DefaultPktRate, DefaultBurstUs, 42, 500)
	for i := range a {
		if a[i].Measured != b[i].Measured {
			t.Fatalf("同种子结果应一致：want %v, got %v", a[i].Measured, b[i].Measured)
		}
	}
}

func TestMissStreakAndOverhead(t *testing.T) {
	if got := MissStreak(0.5, 2); math.Abs(got-0.25) > 1e-9 {
		t.Fatalf("MissStreak(0.5,2) 应为 0.25，got %v", got)
	}
	if got := OverheadPct(1450, 5); math.Abs(got-8.28) > 0.01 {
		t.Fatalf("5 跳相对 1450 字节开销应约 8.28%%，got %.2f", got)
	}
	if got := OverheadPct(0, 5); got != 0 {
		t.Fatalf("零长度报文应返回 0，got %v", got)
	}
}

func TestParseScenario(t *testing.T) {
	if _, err := ParseScenario("egress"); err != nil {
		t.Fatalf("合法场景报错：%v", err)
	}
	if _, err := ParseScenario("nope"); err == nil {
		t.Fatal("非法场景应当报错")
	}
}
