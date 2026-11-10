package main

import (
	"math"
	"sort"
)

// StepStat 是一步集合通信的核算结果。
type StepStat struct {
	Step     int
	MsgBytes float64 // 该步消息大小，从流量反推
	BusyUS   float64 // 步耗时 = 最慢 rank 的传输耗时（同步通信，全环等最慢）
	AlgBW    float64 // 算法带宽 GB/s = 消息大小 / 步耗时
	BusBW    float64 // 总线带宽 GB/s = AlgBW × 2(N-1)/N
	SlowRank int     // 该步最慢的 rank
	Anomaly  bool    // busbw 明显低于基线
}

// Analysis 是整份日志的核算结论。
type Analysis struct {
	N           int
	Ring        []int
	RingFromLog bool // true = 日志自带 Ring 行，false = 从流量反推

	MsgBytes     float64
	PerPeerBytes float64 // 每 rank 每步发给下一跳的累计字节
	Steps        []StepStat

	Matrix     map[[2]int]float64 // (源, 目的) → 每步字节
	Baseline   float64            // 基线 busbw，取各步中位数
	BaselineUS float64            // 基线步耗时

	AnomalySteps []int
	SuspectRank  int // 异常步里出现最多的最慢 rank
}

// Analyze 从解析结果算出逐步带宽、流量矩阵与异常归因。
func Analyze(l *Log) *Analysis {
	a := &Analysis{N: l.NRanks, MsgBytes: l.MsgBytes, Matrix: map[[2]int]float64{}}

	if len(l.Ring) == l.NRanks {
		a.Ring = l.Ring
		a.RingFromLog = true
	} else if r := l.RebuildRing(); r != nil {
		a.Ring = r
	} else {
		a.Ring = make([]int, l.NRanks)
		for i := range a.Ring {
			a.Ring[i] = i
		}
	}

	// 按步聚合
	byStep := map[int][]Send{}
	for _, s := range l.Sends {
		byStep[s.Step] = append(byStep[s.Step], s)
		a.Matrix[[2]int{s.Rank, s.Peer}] += s.Bytes
	}
	steps := make([]int, 0, len(byStep))
	for st := range byStep {
		steps = append(steps, st)
	}
	sort.Ints(steps)

	n := float64(a.N)
	for _, st := range steps {
		sends := byStep[st]
		slow := sends[0]
		total := 0.0
		for _, s := range sends {
			total += s.Bytes
			if s.DurUS > slow.DurUS {
				slow = s
			}
		}
		// 每步所有 rank 的总发送量 = 2(N-1) × S
		msg := total / (2 * (n - 1))
		if msg <= 0 {
			msg = a.MsgBytes
		}
		alg := msg / (slow.DurUS * 1e3) // bytes/μs → GB/s：除以 1e3
		a.Steps = append(a.Steps, StepStat{
			Step:     st,
			MsgBytes: msg,
			BusyUS:   slow.DurUS,
			AlgBW:    alg,
			BusBW:    alg * 2 * (n - 1) / n,
			SlowRank: slow.Rank,
		})
	}
	if len(a.Steps) == 0 {
		return a
	}

	// 每 rank 每步发往下一跳的字节数
	var perPeer []float64
	for _, s := range a.Steps {
		perPeer = append(perPeer, s.MsgBytes*2*(n-1)/n)
	}
	a.PerPeerBytes = median(perPeer)

	// 基线取 busbw 中位数：少数故障步拉不动它
	bws := make([]float64, len(a.Steps))
	uss := make([]float64, len(a.Steps))
	for i, s := range a.Steps {
		bws[i] = s.BusBW
		uss[i] = s.BusyUS
	}
	a.Baseline = median(bws)
	a.BaselineUS = median(uss)

	// 低于基线 75% 判为异常
	count := map[int]int{}
	for i := range a.Steps {
		if a.Steps[i].BusBW < a.Baseline*0.75 {
			a.Steps[i].Anomaly = true
			a.AnomalySteps = append(a.AnomalySteps, a.Steps[i].Step)
			count[a.Steps[i].SlowRank]++
		}
	}
	best, bestN := -1, 0
	for r, c := range count {
		if c > bestN || (c == bestN && r < best) {
			best, bestN = r, c
		}
	}
	a.SuspectRank = best
	return a
}

// WastedRankUS 返回异常步相比基线多花的时间，乘以卡数得到被浪费的卡·μs。
func (a *Analysis) WastedRankUS() float64 {
	wasted := 0.0
	for _, s := range a.Steps {
		if s.Anomaly {
			wasted += (s.BusyUS - a.BaselineUS) * float64(a.N)
		}
	}
	return wasted
}

// Efficiency 返回基线 busbw 相对端口的效率。portGBps 未知时按 25 GB/s 估。
func (a *Analysis) Efficiency(portGBps float64) float64 {
	if portGBps <= 0 {
		return 0
	}
	return math.Min(1, a.Baseline/portGBps)
}

func median(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	s := append([]float64(nil), vals...)
	sort.Float64s(s)
	n := len(s)
	if n%2 == 1 {
		return s[n/2]
	}
	return (s[n/2-1] + s[n/2]) / 2
}
