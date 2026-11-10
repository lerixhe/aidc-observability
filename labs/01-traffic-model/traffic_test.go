package main

import (
	"math"
	"math/rand"
	"testing"
)

// 两种模型的平均负载必须对齐，否则后面的对比没有意义。
func TestMeansMatch(t *testing.T) {
	p := DefaultParams()
	idc := idcTraffic(p)
	ar := allreduceTraffic(p)

	if math.Abs(idc.Mean()-p.MeanGBps) > p.MeanGBps*0.08 {
		t.Errorf("idc 平均负载 %.1f GB/s 偏离目标 %.1f 太多", idc.Mean(), p.MeanGBps)
	}
	if math.Abs(ar.Mean()-p.MeanGBps) > p.MeanGBps*0.02 {
		t.Errorf("allreduce 平均负载 %.1f GB/s 偏离目标 %.1f 太多", ar.Mean(), p.MeanGBps)
	}
}

// 这是本 lab 的核心断言：轮询窗口越长，同步突发的峰值被抹得越平。
func TestPolledPeakShrinksWithWindow(t *testing.T) {
	p := DefaultParams()
	ar := allreduceTraffic(p)

	prev := math.Inf(1)
	for _, win := range []int{1, 10, 100, 1000} {
		got := PolledPeak(ar.GBps, win)
		if got > prev {
			t.Errorf("窗口 %d ms 的观测峰值 %.1f 比更小窗口的 %.1f 还高", win, got, prev)
		}
		prev = got
	}
	if peak := PolledPeak(ar.GBps, 1000); peak > p.PortGBps {
		t.Errorf("1 s 窗口下峰值 %.1f GB/s 仍高于线速 %.1f，同步突发没被抹平", peak, p.PortGBps)
	}
}

// 传统 IDC 的峰值受端口线速约束，不可能出现倍数级放大。
func TestIDCNeverExceedsLineRate(t *testing.T) {
	p := DefaultParams()
	idc := idcTraffic(p)
	if hi := maxOf(idc.GBps); hi > p.PortGBps+1e-9 {
		t.Errorf("idc 峰值 %.2f GB/s 超过端口线速 %.2f，摊平逻辑有问题", hi, p.PortGBps)
	}
}

func TestQuantile(t *testing.T) {
	sorted := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	cases := map[float64]float64{0: 1, 0.5: 5, 0.9: 9, 1: 10}
	for q, want := range cases {
		if got := quantile(sorted, q); got != want {
			t.Errorf("quantile(%v) = %v，期望 %v", q, got, want)
		}
	}
}

// 对数正态的采样均值要贴近解析均值 median*exp(sigma²/2)。
func TestLogNormalMean(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	const n = 200000
	median := 0.000064 // 64 KB
	sigma := 1.5
	sum := 0.0
	for i := 0; i < n; i++ {
		sum += logNormal(rng, median, sigma)
	}
	got := sum / n
	want := median * math.Exp(sigma*sigma/2)
	if math.Abs(got-want)/want > 0.05 {
		t.Errorf("采样均值 %.3g 与解析均值 %.3g 偏差过大", got, want)
	}
}

func TestValidateRejectsBadParams(t *testing.T) {
	p := DefaultParams()
	p.Workers = 0
	if err := validate(p); err == nil {
		t.Error("workers=0 应当被拒绝")
	}

	p = DefaultParams()
	p.StepMS = 100000
	if err := validate(p); err == nil {
		t.Error("step 周期大于观测时长应当被拒绝")
	}

	p = DefaultParams()
	p.MeanGBps = 500 // 通信期会超过整个周期
	if err := validate(p); err == nil {
		t.Error("平均负载高到通信阶段撑爆周期，应当被拒绝")
	}
}

func TestDeterministic(t *testing.T) {
	p := DefaultParams()
	p.DurationMS = 2000
	a := idcTraffic(p)
	b := idcTraffic(p)
	for i := range a.GBps {
		if a.GBps[i] != b.GBps[i] {
			t.Fatalf("同种子第 %d ms 结果不一致：%v vs %v", i, a.GBps[i], b.GBps[i])
		}
	}
}
