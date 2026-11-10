package main

import (
	"math"
	"math/rand"
)

// 本文件是「遥测指纹模型」：给定故障注入方式，生成该端口在观测窗口内的
// 计数器时序。它不是网络仿真，只是把第 07 课分诊表里的判据写成可计算的形态。
//
// 单位约定：
//
//	时间  秒
//	时延  ms，-1 表示超时（探测无响应）
//	丢包  %
//	pause 该端口在一个采样周期里处于反压状态的时间占比，%
//	ECN   被标记报文的占比，%
//	队列  端口出队列水位，KB

// profile 描述单个端口在窗口内的形态：一条基线，加上劣化后的增量。
//
// 同一套字段能表达四类故障，靠的是组合方式不同：
//
//	风暴    多条端口 pause 大幅抬升、零丢包、时延持续高
//	死锁    pause 顶满、时延超时、零丢包、ECN 归零（不流动了）
//	incast  周期型尖刺，丢包非零、ECN 尖刺、队列周期性顶高
//	队头    少数端口时延温和抬升，pause 与 ECN 都不高
type profile struct {
	Port string
	Role string // culprit 肇事 / victim 受害 / healthy 健康对照

	LatP50, LatP99 float64 // 基线时延，ms
	LatMul         float64 // 劣化后的时延倍数，0 表示直接超时

	LossBase, LossStep   float64
	PauseBase, PauseStep float64
	ECNBase, ECNStep     float64
	QBase, QStep         float64

	CRCStep        float64 // CRC 每分钟增量，从 0 线性爬升到该值
	FECFrom, FECTo float64 // pre-FEC BER 起止值

	Period float64 // 周期尖刺的周期，秒；0 表示劣化后持续
	Duty   float64 // 周期内处于尖刺状态的比例，1 表示全程

	Jitter float64 // 抖动幅度，占当前值的比例
}

// activeAt 返回 t 时刻端口的劣化强度：0 表示基线，1 表示完全劣化。
func (p profile) activeAt(t, onset float64) float64 {
	if t < onset {
		return 0
	}
	if p.Period <= 0 {
		return 1
	}
	phase := math.Mod(t-onset, p.Period)
	if phase < p.Period*p.Duty {
		return 1
	}
	return 0
}

// Row 是样本文件里的一行：某端口在某一秒的观测值。
type Row struct {
	Port      string
	Role      string
	T         float64
	LatP50    float64
	LatP99    float64
	LossPct   float64
	PausePct  float64
	ECNPct    float64
	QDepthKB  float64
	CRCPerMin float64
	FECBER    float64
}

// Degraded 判断这一行看起来是否异常，供分类器统计受影响端口数。
func (r Row) Degraded() bool {
	if r.LatP99 < 0 {
		return true
	}
	return r.PausePct >= 3 || r.LossPct >= 0.1 || r.ECNPct >= 2
}

// simulatePort 生成单个端口在窗口内的采样序列。
func simulatePort(p profile, window, step, onset float64, rng *rand.Rand) []Row {
	n := int(math.Round(window / step))
	if n < 1 {
		n = 1
	}
	rows := make([]Row, 0, n)
	for i := 0; i < n; i++ {
		t := float64(i) * step
		a := p.activeAt(t, onset)

		// 抖动是乘性的：基线本身也有小波动，否则样本过于光滑，判读训练没意义
		jit := func() float64 {
			if p.Jitter == 0 {
				return 1
			}
			return 1 + rng.NormFloat64()*p.Jitter
		}

		r := Row{Port: p.Port, Role: p.Role, T: t}
		if p.LatMul == 0 && a > 0 {
			r.LatP50, r.LatP99 = -1, -1
		} else {
			mul := 1 + (p.LatMul-1)*a
			r.LatP50 = p.LatP50 * mul * jit()
			r.LatP99 = p.LatP99 * mul * jit()
		}
		r.LossPct = clamp(p.LossBase+p.LossStep*a*jit(), 0, 100)
		r.PausePct = clamp(p.PauseBase+p.PauseStep*a*jit(), 0, 100)
		r.ECNPct = clamp(p.ECNBase+p.ECNStep*a*jit(), 0, 100)
		r.QDepthKB = clamp(p.QBase+p.QStep*a*jit(), 0, math.MaxInt32)
		r.CRCPerMin = clamp(p.CRCStep*(t/window)*jit(), 0, 1e9)
		r.FECBER = p.FECFrom + (p.FECTo-p.FECFrom)*(t/window)
		rows = append(rows, r)
	}
	return rows
}

// Simulate 把一个场景的所有端口序列拼成一份样本。
func Simulate(s Scenario) []Row {
	rng := rand.New(rand.NewSource(s.Seed))
	out := make([]Row, 0, len(s.Ports)*int(s.Window/s.Step))
	for _, p := range s.Ports {
		out = append(out, simulatePort(p, s.Window, s.Step, s.Onset*s.Window, rng)...)
	}
	return out
}

func clamp(v, lo, hi float64) float64 {
	return math.Min(math.Max(v, lo), hi)
}
