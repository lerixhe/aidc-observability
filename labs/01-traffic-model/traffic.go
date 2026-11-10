package main

import (
	"math"
	"math/rand"
)

// 单位约定（整个 lab 统一）
//
//	时间：ms
//	数据量：GB
//	速率：GB/s
//
// 一个 200 Gbps 端口 = 25 GB/s；4 条这样的端口汇聚到一个出端口，
// 通信期瞬时到达率 100 GB/s。

// Params 是两种流量模型的参数。
type Params struct {
	DurationMS int     // 观测时长，ms
	MeanGBps   float64 // 目标平均负载，GB/s，两种模型必须对齐
	PortGBps   float64 // 单端口线速，GB/s
	Workers    int     // AllReduce 的并发 worker 数

	// 传统 IDC：大量小流随机到达，流大小服从对数正态
	FlowMedianKB float64 // 流大小中位数，KB
	FlowSigma    float64 // 对数正态的 sigma，控制长尾

	// AllReduce：按训练步节奏周期性同步突发
	StepMS int // 一个训练步的周期，ms
	Seed   int64
}

// DefaultParams 给出「同平均负载、只差到达形态」的一组对照参数。
func DefaultParams() Params {
	return Params{
		DurationMS:   20000, // 20 s，够看清 100 个训练步
		MeanGBps:     20,
		PortGBps:     25,
		Workers:      4,
		FlowMedianKB: 64,
		FlowSigma:    1.5,
		StepMS:       200,
		Seed:         42,
	}
}

// Traffic 是一段按 1 ms 分辨率采样的到达率序列。
type Traffic struct {
	Name   string
	GBps   []float64 // 每 1 ms 的瞬时到达率
	Note   string    // 一句话描述它的节奏
	CommMS float64   // AllReduce 的通信阶段长度，IDC 为 0
	Lambda float64   // IDC 的每 ms 流数，AllReduce 为 0
}

// Mean 返回实际平均负载。
func (t *Traffic) Mean() float64 {
	sum := 0.0
	for _, v := range t.GBps {
		sum += v
	}
	if len(t.GBps) == 0 {
		return 0
	}
	return sum / float64(len(t.GBps))
}

// flowMeanGB 返回对数正态流大小的均值。
// 中位数 m、sigma s 时，均值 = m * exp(s²/2)。
func (p *Params) flowMeanGB() float64 {
	return p.FlowMedianKB / 1e6 * math.Exp(p.FlowSigma*p.FlowSigma/2)
}

// idcTraffic 用泊松到达 + 对数正态流大小生成传统 IDC 流量。
//
// 每条流都要真切地流过端口，所以加了一个「按线速摊平」的过程：
// 一笔大流不会在 1 ms 里凭空出现，它会把后续几 ms 也占满。
// 这就是传统 IDC 看不到倍数级突发的原因 —— 端口线速是硬天花板。
func idcTraffic(p Params) Traffic {
	rng := rand.New(rand.NewSource(p.Seed))
	meanFlowGB := p.flowMeanGB()
	// MeanGBps 是 GB/s，除以 1000 换成 GB/ms，再除以单流均值得到每 ms 的流数
	lambda := p.MeanGBps / 1000 / meanFlowGB
	gbps := make([]float64, p.DurationMS)
	pending := 0.0 // 已到达但还没发完的数据，GB

	for ms := 0; ms < p.DurationMS; ms++ {
		n := poisson(rng, lambda)
		for i := 0; i < n; i++ {
			pending += logNormal(rng, p.FlowMedianKB/1e6, p.FlowSigma)
		}
		// 1 ms 内一个端口最多送出 PortGBps*0.001 GB
		budget := p.PortGBps * 0.001
		out := math.Min(pending, budget)
		pending -= out
		gbps[ms] = out * 1000 // GB/ms 换算回 GB/s
	}
	return Traffic{
		Name:   "idc 小流",
		GBps:   gbps,
		Note:   "泊松到达 + 对数正态流大小，但每条流都受端口线速约束",
		Lambda: lambda,
	}
}

// logNormal 按「中位数 + sigma」采样一个流大小（GB）。
func logNormal(rng *rand.Rand, medianGB, sigma float64) float64 {
	return medianGB * math.Exp(sigma*rng.NormFloat64())
}

// poisson 用 Knuth 算法采样泊松分布。lambda 上百时用正态近似更快。
func poisson(rng *rand.Rand, lambda float64) int {
	if lambda <= 0 {
		return 0
	}
	if lambda > 100 {
		v := rng.NormFloat64()*math.Sqrt(lambda) + lambda
		if v < 0 {
			return 0
		}
		return int(v + 0.5)
	}
	l := math.Exp(-lambda)
	k, p := 0, 1.0
	for {
		p *= rng.Float64()
		if p <= l {
			return k
		}
		k++
	}
}

// allreduceTraffic 按训练步节奏生成同步突发：每 step 里只有一小段在通信，
// 通信期所有 worker 同时朝同一目的地全速发。
func allreduceTraffic(p Params) Traffic {
	burstRate := float64(p.Workers) * p.PortGBps         // 通信期瞬时到达率
	commMS := float64(p.StepMS) * p.MeanGBps / burstRate // 反推通信阶段长度
	gbps := make([]float64, p.DurationMS)

	for ms := 0; ms < p.DurationMS; ms++ {
		phase := float64(ms % p.StepMS)
		if phase < commMS {
			gbps[ms] = burstRate
		} else {
			gbps[ms] = 0
		}
	}
	return Traffic{
		Name:   "AllReduce 同步突发",
		GBps:   gbps,
		Note:   "每周期集中全速发一小段，其余时间算梯度",
		CommMS: commMS,
	}
}
