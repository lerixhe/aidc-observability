package main

import (
	"math"
	"testing"
)

func TestGenerateShape(t *testing.T) {
	pts := Generate(1)
	if len(pts) != MinutesPerDay {
		t.Fatalf("一天应有 %d 个窗口，got %d", MinutesPerDay, len(pts))
	}
	for i, s := range pts {
		if s.TsMin != i {
			t.Fatalf("时间戳应连续，第 %d 个是 %d", i, s.TsMin)
		}
		if s.Reqs <= 0 {
			t.Fatalf("请求数必须为正：t=%d", s.TsMin)
		}
		if s.Bad > s.Reqs {
			t.Fatalf("失败数不能超过请求数：t=%d %d>%d", s.TsMin, s.Bad, s.Reqs)
		}
	}
}

func TestGenerateTailIncident(t *testing.T) {
	pts := Generate(DefaultDays)
	last := pts[len(pts)-1]
	if last.FailRate() < 0.1 {
		t.Fatalf("窗口末尾应处于正在发生的事故里，失败率 %v", last.FailRate())
	}
	if IncidentAt(last.TsMin) == nil {
		t.Fatal("末尾窗口应命中事故清单")
	}
	if IncidentAt(0) != nil {
		t.Fatal("起点不该命中任何事故")
	}
}

func TestTrafficRateDayNight(t *testing.T) {
	day := trafficRate(12 * 60) // 12:00
	night := trafficRate(2 * 60) // 02:00
	if day <= night {
		t.Fatalf("白天流量应大于夜间：day=%v night=%v", day, night)
	}
}

func mkSample(ts, reqs, bad int, lat float64) Sample {
	return Sample{TsMin: ts, Reqs: reqs, Bad: bad, LatMs: lat}
}

func TestSampleBadMinute(t *testing.T) {
	cases := []struct {
		name string
		s    Sample
		want bool
	}{
		{"健康", mkSample(0, 1000, 1, 8), false},
		{"失败率超线", mkSample(0, 1000, 20, 8), true},
		{"时延超线", mkSample(0, 1000, 1, 60), true},
		{"空窗口", Sample{}, false},
	}
	for _, c := range cases {
		if got := c.s.BadMinute(); got != c.want {
			t.Errorf("%s：want %v, got %v", c.name, c.want, got)
		}
	}
}

// 构造一组数据：夜间半小时全坏但流量极低，白天一分钟坏但流量高。
// 时间口径会算成「两段都坏」，事件口径则几乎只看到白天那一段。
func nightVsDaySamples() []Sample {
	var out []Sample
	for t := 0; t < 60; t++ {
		if t < 30 {
			out = append(out, mkSample(t, 100, 30, 200)) // 夜间 30 分钟，低流量，30% 失败
		} else {
			out = append(out, mkSample(t, 10000, 1, 8)) // 白天 30 分钟，高流量，基本正常
		}
	}
	return out
}

func TestComputeCalibersDiverge(t *testing.T) {
	samples := nightVsDaySamples()
	timeR := Compute(CaliberTime, samples, SLOTarget)
	eventR := Compute(CaliberEvent, samples, SLOTarget)
	latR := Compute(CaliberLatency, samples, SLOTarget)

	// 时间口径：30/60 的分钟是坏的
	if math.Abs(timeR.Availability-0.5) > 1e-9 {
		t.Fatalf("时间口径应为 0.5，got %v", timeR.Availability)
	}
	// 事件口径：坏请求 30*30=900 加白天每分钟 1 条共 30 条 = 930，
	// 总请求 30*100+30*10000=303000
	wantEvent := 1 - 930.0/303000.0
	if math.Abs(eventR.Availability-wantEvent) > 1e-9 {
		t.Fatalf("事件口径应为 %v，got %v", wantEvent, eventR.Availability)
	}
	if eventR.Availability <= timeR.Availability {
		t.Fatalf("低流量事故下事件口径应更宽：time=%v event=%v", timeR.Availability, eventR.Availability)
	}
	// 分位口径只看时延：30 分钟超 50ms
	if math.Abs(latR.Availability-0.5) > 1e-9 {
		t.Fatalf("分位口径应为 0.5，got %v", latR.Availability)
	}
	if latR.BadUnits != 30 {
		t.Fatalf("分位口径坏单位应为 30 分钟，got %d", latR.BadUnits)
	}
}

func TestComputeLatencyIgnoresErrors(t *testing.T) {
	// 全部失败但时延很低：分位口径判「达标」，事件口径判「未达标」
	samples := make([]Sample, 30)
	for i := range samples {
		samples[i] = mkSample(i, 1000, 1000, 5)
	}
	latR := Compute(CaliberLatency, samples, SLOTarget)
	eventR := Compute(CaliberEvent, samples, SLOTarget)
	if !latR.Met {
		t.Fatalf("只看时延的口径应判达标，got %v", latR.Availability)
	}
	if eventR.Met {
		t.Fatalf("全失败时事件口径应判未达标，got %v", eventR.Availability)
	}
}

func TestComputeMetAndGap(t *testing.T) {
	samples := []Sample{}
	for i := 0; i < 1000; i++ {
		samples = append(samples, mkSample(i, 1000, 0, 8))
	}
	r := Compute(CaliberEvent, samples, SLOTarget)
	if r.Availability != 1 {
		t.Fatalf("无失败时应为 1，got %v", r.Availability)
	}
	if !r.Met || r.Verdict() != "达标" {
		t.Fatalf("应判达标：%v %s", r.Met, r.Verdict())
	}
	if got := r.GapBp(SLOTarget); math.Abs(got-10) > 1e-9 {
		t.Fatalf("超出 0.1%% 应为 10 个基点，got %v", got)
	}
}

func TestComputeEmpty(t *testing.T) {
	for _, c := range Calibers {
		r := Compute(c, nil, SLOTarget)
		if r.TotalUnits != 0 {
			t.Errorf("%s 空输入总单位应为 0，got %d", c, r.TotalUnits)
		}
	}
}

func TestCompareNonEmpty(t *testing.T) {
	samples := nightVsDaySamples()
	rs := make([]Result, 0, len(Calibers))
	for _, c := range Calibers {
		rs = append(rs, Compute(c, samples, SLOTarget))
	}
	if got := Compare(rs); got == "" {
		t.Fatal("对比结论不该为空")
	}
	if got := Compare(rs[:1]); got == "" {
		t.Fatal("单口径也应给出提示而不是空串")
	}
}

func TestTimeBudget(t *testing.T) {
	samples := make([]Sample, 1000)
	for i := range samples {
		samples[i] = mkSample(i, 1000, 0, 8)
	}
	b := TimeBudget(samples, 0.999)
	if math.Abs(b.Allowed-1.0) > 1e-9 {
		t.Fatalf("1000 分钟 99.9%% 应允许 1 分钟，got %v", b.Allowed)
	}
	if b.Consumed != 0 || b.Exhausted() {
		t.Fatalf("无坏分钟时不该耗尽：%+v", b)
	}
	if math.Abs(b.Remaining-1.0) > 1e-9 {
		t.Fatalf("剩余应等于允许额度，got %v", b.Remaining)
	}
}

func TestEventBudgetExhausted(t *testing.T) {
	samples := make([]Sample, 100)
	for i := range samples {
		samples[i] = mkSample(i, 1000, 100, 200) // 10% 失败
	}
	b := EventBudget(samples, 0.999)
	// 允许 100000*0.001=100 条，实际消耗 10000 条
	if !b.Exhausted() {
		t.Fatalf("预算应已耗尽：%+v", b)
	}
	if b.UsedPercent < 99 {
		t.Fatalf("已用比例应远超 100%%，got %v", b.UsedPercent)
	}
}

func TestBurnRateMath(t *testing.T) {
	// 全坏窗口：burn = 1/0.001 = 1000
	allBad := make([]Sample, 10)
	for i := range allBad {
		allBad[i] = mkSample(i, 1000, 1000, 200)
	}
	if got := BurnRate(allBad, 10, 0.999); math.Abs(got-1000) > 1e-6 {
		t.Fatalf("全坏窗口燃烧率应为 1000，got %v", got)
	}

	// 完全健康：燃烧率 0
	healthy := make([]Sample, 10)
	for i := range healthy {
		healthy[i] = mkSample(i, 1000, 0, 8)
	}
	if got := BurnRate(healthy, 10, 0.999); got != 0 {
		t.Fatalf("健康窗口燃烧率应为 0，got %v", got)
	}
	if got := BurnRate(nil, 10, 0.999); got != 0 {
		t.Fatalf("空数据燃烧率应为 0，got %v", got)
	}
}

func TestTail(t *testing.T) {
	samples := Generate(1)
	if got := Tail(samples, 10); len(got) != 10 {
		t.Fatalf("应取最后 10 个，got %d", len(got))
	}
	if got := Tail(samples, 0); len(got) != len(samples) {
		t.Fatalf("窗口为 0 时应返回全量，got %d", len(got))
	}
	if got := Tail(samples, len(samples)+10); len(got) != len(samples) {
		t.Fatalf("窗口超过长度时应返回全量，got %d", len(got))
	}
}

func TestMultiWindowBurnPaging(t *testing.T) {
	samples := Generate(DefaultDays)
	rows, fast := MultiWindowBurn(samples, SLOTarget)
	if len(rows) != 4 {
		t.Fatalf("应返回四个窗口，got %d", len(rows))
	}
	if !fast {
		t.Fatalf("窗口末尾有正在发生的事故，快烧两窗口应同时超阈值：%+v", rows)
	}
	// 长窗口（6 小时）不该被 10 分钟的末尾事故打爆
	if rows[3].Firing {
		t.Fatalf("6 小时窗口不该超阈值：%+v", rows[3])
	}
}

func TestTopMinutes(t *testing.T) {
	samples := Generate(DefaultDays)
	top := TopMinutes(samples, 3)
	if len(top) != 3 {
		t.Fatalf("应返回 3 条，got %d", len(top))
	}
	for i := 1; i < len(top); i++ {
		if top[i].FailRate() > top[i-1].FailRate() {
			t.Fatal("结果应按失败率降序")
		}
	}
	if got := TopMinutes(samples, 10*len(samples)); len(got) != len(samples) {
		t.Fatalf("n 超界时应返回全量，got %d", len(got))
	}
}

func TestWindowImpact(t *testing.T) {
	samples := Generate(DefaultDays)
	in := Incidents[0]
	bads, badReqs := windowImpact(samples, in.FromMin, in.ToMin)
	if bads != in.ToMin-in.FromMin {
		t.Fatalf("事故期间每个分钟都该判坏：want %d, got %d", in.ToMin-in.FromMin, bads)
	}
	if badReqs == 0 {
		t.Fatal("应有坏请求")
	}
}
