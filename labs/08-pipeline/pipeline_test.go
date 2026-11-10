package main

import (
	"math"
	"testing"
)

func TestGenerateShape(t *testing.T) {
	pts := Generate(600)
	if len(pts) != 600 {
		t.Fatalf("点数应为 600，got %d", len(pts))
	}
	for i, p := range pts {
		if p.TsSec != i {
			t.Fatalf("时间戳应从 0 连续递增，第 %d 个是 %d", i, p.TsSec)
		}
		if p.KB < 0 {
			t.Fatalf("队列占用不该为负：t=%d got %v", p.TsSec, p.KB)
		}
	}
}

func TestGenerateHasInjectedSpikes(t *testing.T) {
	pts := Generate(DefaultWindowSec)
	peak := 0.0
	healthyPeak := 0.0
	for _, p := range pts {
		if p.KB > peak {
			peak = p.KB
		}
		if p.TsSec < DangerFromSec && p.KB > healthyPeak {
			healthyPeak = p.KB
		}
	}
	if peak != 520.0 {
		t.Fatalf("劣化窗口应注入 520 KB 的冲顶，got %v", peak)
	}
	if healthyPeak >= QueueThresholdKB {
		t.Fatalf("健康期不该越过阈值，got %v", healthyPeak)
	}
	if got := len(DangerPoints(pts)); got != DangerToSec-DangerFromSec {
		t.Fatalf("劣化窗口点数应为 %d，got %d", DangerToSec-DangerFromSec, got)
	}
}

func TestQuantile(t *testing.T) {
	vals := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	if got := quantile(vals, 0.99); got != 10 {
		t.Fatalf("最近秩法 p99 应取到 10，got %v", got)
	}
	if got := quantile(vals, 0.5); got != 5 {
		t.Fatalf("最近秩法 p50 应取到 5，got %v", got)
	}
	if got := quantile(nil, 0.99); got != 0 {
		t.Fatalf("空集应返回 0，got %v", got)
	}
	// 排序不应改动入参
	if vals[0] != 1 || vals[9] != 10 {
		t.Fatalf("quantile 不该修改入参：%v", vals)
	}
}

func TestAggregateKeepPeak(t *testing.T) {
	pts := []Point{{0, 10}, {1, 20}, {2, 500}, {3, 30}}
	withPeak := Aggregate(pts, 0, 4, true)
	if withPeak.Count != 4 {
		t.Fatalf("count 应为 4，got %d", withPeak.Count)
	}
	if withPeak.Max != 500 {
		t.Fatalf("max 应为 500，got %v", withPeak.Max)
	}
	if withPeak.Mean != 140 {
		t.Fatalf("mean 应为 140，got %v", withPeak.Mean)
	}

	withoutPeak := Aggregate(pts, 0, 4, false)
	if withoutPeak.Max != 0 || withoutPeak.P99 != 0 {
		t.Fatalf("不保留峰值时 max/p99 应为 0，got %v/%v", withoutPeak.Max, withoutPeak.P99)
	}
	if withoutPeak.Mean != withPeak.Mean {
		t.Fatalf("两种口径的均值应一致")
	}
}

func TestAggregateEmptyWindow(t *testing.T) {
	b := Aggregate([]Point{{0, 10}}, 100, 200, true)
	if b.Count != 0 || b.Mean != 0 {
		t.Fatalf("空窗口应返回零值桶，got %+v", b)
	}
}

func TestBuildSeriesBucketCounts(t *testing.T) {
	pts := Generate(DefaultWindowSec) // 3600 点
	cases := []struct {
		tier  Tier
		want  int
		width int
	}{
		{TierRaw, 3600, 1},
		{TierMinuteMax, 60, 60},
		{TierMinuteMean, 60, 60},
		{TierDailyMax, 1, 86400},
	}
	for _, c := range cases {
		s := BuildSeries(c.tier, pts)
		if len(s.Buckets) != c.want {
			t.Errorf("%s 桶数应为 %d，got %d", c.tier, c.want, len(s.Buckets))
		}
		if s.PointsPerBucket != c.width {
			t.Errorf("%s 每桶点数应为 %d，got %d", c.tier, c.width, s.PointsPerBucket)
		}
	}

	raw := BuildSeries(TierRaw, pts)
	minute := BuildSeries(TierMinuteMax, pts)
	if raw.BucketBytes() <= minute.BucketBytes()*50 {
		t.Fatalf("原始档体积应显著大于分钟档：raw=%d minute=%d", raw.BucketBytes(), minute.BucketBytes())
	}
}

func TestQueryWindowMeanOnlyLosesPeak(t *testing.T) {
	pts := Generate(DefaultWindowSec)
	from, to := DangerFromSec, DangerToSec

	raw := QueryWindow(BuildSeries(TierRaw, pts), from, to)
	maxed := QueryWindow(BuildSeries(TierMinuteMax, pts), from, to)
	meanOnly := QueryWindow(BuildSeries(TierMinuteMean, pts), from, to)

	if raw.PeakKB != 520 {
		t.Fatalf("原始档应看到 520 KB 峰值，got %v", raw.PeakKB)
	}
	if maxed.PeakKB != 520 {
		t.Fatalf("minutes(max) 应保住峰值 520，got %v", maxed.PeakKB)
	}
	if meanOnly.PeakKB >= QueueThresholdKB {
		t.Fatalf("只留均值的档位不该越过阈值，got %v", meanOnly.PeakKB)
	}
	if raw.Alarmed != true || meanOnly.Alarmed != false {
		t.Fatalf("告警判定应分化：raw=%v meanOnly=%v", raw.Alarmed, meanOnly.Alarmed)
	}
	if math.Abs(raw.MeanKB-maxed.MeanKB) > 0.1 {
		t.Fatalf("原始档与分钟档的均值应一致：%v vs %v", raw.MeanKB, maxed.MeanKB)
	}
}

func TestQueryWindowOutsideRange(t *testing.T) {
	pts := Generate(600)
	a := QueryWindow(BuildSeries(TierRaw, pts), 5000, 6000)
	if a.Buckets != 0 || a.PeakKB != 0 || a.Alarmed {
		t.Fatalf("窗口外应返回空结果，got %+v", a)
	}
}

func TestSimulateIngestSteadyState(t *testing.T) {
	pts := Generate(120) // 前 120 秒没有降档
	res := SimulateIngest(pts, DefaultBufferCap)
	if res.Dropped != 0 {
		t.Fatalf("稳态不该丢点，got %d", res.Dropped)
	}
	if res.Completeness() != 1 {
		t.Fatalf("稳态完整率应为 1，got %v", res.Completeness())
	}
	if res.FirstDropAt != -1 {
		t.Fatalf("没有丢弃时 FirstDropAt 应为 -1，got %d", res.FirstDropAt)
	}
}

func TestSimulateIngestDropsDuringSlowdown(t *testing.T) {
	pts := Generate(DefaultWindowSec)
	res := SimulateIngest(pts, DefaultBufferCap)

	if res.Dropped == 0 {
		t.Fatal("降档 300 秒且缓冲 200 点，应当出现丢弃")
	}
	if res.FirstDropAt < SlowStartSec {
		t.Fatalf("首次丢弃不该早于降档起点：got t=%d", res.FirstDropAt)
	}
	if res.PeakBuffer > res.BufferCap {
		t.Fatalf("峰值缓冲不该超过容量：%d > %d", res.PeakBuffer, res.BufferCap)
	}
	if res.Completeness() >= 1 {
		t.Fatalf("有丢弃时完整率应小于 1，got %v", res.Completeness())
	}
	// 健康窗口必须完整：这正是「面板看起来一切正常」的来源
	if got := res.WindowCompleteness(0, SlowStartSec); got != 1 {
		t.Fatalf("健康窗口完整率应为 1，got %v", got)
	}
	if got := res.WindowCompleteness(DangerFromSec, DangerToSec); got >= 1 {
		t.Fatalf("事故窗口完整率应明显低于 1，got %v", got)
	}
	if got := res.DroppedInWindow(DangerFromSec, DangerToSec); got != res.Dropped {
		t.Fatalf("丢弃应集中在事故窗口内：窗口 %d，总 %d", got, res.Dropped)
	}
}

func TestSimulateIngestLargeBufferNoDrop(t *testing.T) {
	pts := Generate(DefaultWindowSec)
	res := SimulateIngest(pts, 5000)
	if res.Dropped != 0 {
		t.Fatalf("缓冲足够大时不该丢点，got %d", res.Dropped)
	}
}

func TestSimulateIngestZeroCapUsesDefault(t *testing.T) {
	res := SimulateIngest(Generate(200), 0)
	if res.BufferCap != DefaultBufferCap {
		t.Fatalf("零容量应回落到默认值 %d，got %d", DefaultBufferCap, res.BufferCap)
	}
}

func TestRound1(t *testing.T) {
	if got := round1(3.14159); got != 3.1 {
		t.Fatalf("round1(3.14159) 应为 3.1，got %v", got)
	}
	if got := round1(2.05); got != 2.1 && got != 2.0 {
		t.Fatalf("round1(2.05) 结果异常：%v", got)
	}
}
