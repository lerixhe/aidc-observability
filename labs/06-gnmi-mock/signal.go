package main

import "time"

// Signal 描述一台被观测设备的指标随「虚拟时间」的变化。
//
// 全部用解析式表达：任何时刻都能算出精确值，所以真值（ground truth）与
// 采样结果可以对账 —— 这是本 lab 能算「保真度」的前提。
type Signal struct {
	StormPeriod   time.Duration // PFC 风暴周期
	StormStart    time.Duration // 周期内的起始偏移
	StormLen      time.Duration // 风暴持续时长
	StormPausePPS float64       // 风暴期 pause 帧速率，帧/秒
	StormECNPPS   float64       // 风暴期 ECN 标记速率，个/秒
	BaseGBps      float64       // 常态出方向速率，GB/s
	FlapPeriod    time.Duration // 接口抖动周期
	FlapLen       time.Duration // 抖动持续时长
	FlapAt        time.Duration // 周期内的起始偏移
}

// DefaultSignal 给出一份「常态平稳 + 周期性 PFC 风暴」的信号。
// 风暴 1.5 s、每 20 s 一次，正好卡在秒级采样与十秒级采样的分辨率之间。
func DefaultSignal() *Signal {
	return &Signal{
		StormPeriod:   20 * time.Second,
		StormStart:    10 * time.Second,
		StormLen:      1500 * time.Millisecond,
		StormPausePPS: 400,
		StormECNPPS:   250,
		BaseGBps:      20,
		FlapPeriod:    7 * time.Second,
		FlapLen:       300 * time.Millisecond,
		FlapAt:        3 * time.Second,
	}
}

// Snapshot 是某一时刻的指标快照。计数器是累积值，与真机语义一致。
type Snapshot struct {
	OutOctets  uint64
	OutPFC     uint64
	ECNMarked  uint64
	Discards   uint64
	WatermarkB uint64 // 队列水线，字节
	OperStatus string
	At         time.Duration // 该快照对应的时刻，便于报告引用
}

// At 渲染 t 时刻的快照。
func (s *Signal) At(t time.Duration) Snapshot {
	if t < 0 {
		t = 0
	}
	return Snapshot{
		OutOctets:  uint64(s.baseOctets(t)),
		OutPFC:     uint64(accumulate(t, s.StormPeriod, s.StormStart, s.StormLen, s.StormPausePPS)),
		ECNMarked:  uint64(accumulate(t, s.StormPeriod, s.StormStart, s.StormLen, s.StormECNPPS)),
		Discards:   uint64(accumulate(t, s.StormPeriod, s.StormStart, s.StormLen, s.StormPausePPS*0.02)),
		WatermarkB: s.watermark(t),
		OperStatus: s.operStatus(t),
		At:         t,
	}
}

// PauseRatePPS 返回 t 时刻的真实 pause 帧速率（帧/秒），用来做保真度对账。
func (s *Signal) PauseRatePPS(t time.Duration) float64 {
	if s.inStorm(t) {
		return s.StormPausePPS
	}
	return 0
}

// TruePeakPauseRate 返回整段信号里的真实速率峰值。
func (s *Signal) TruePeakPauseRate() float64 { return s.StormPausePPS }

func (s *Signal) inStorm(t time.Duration) bool {
	rem := t % s.StormPeriod
	return rem >= s.StormStart && rem < s.StormStart+s.StormLen
}

func (s *Signal) baseOctets(t time.Duration) float64 {
	return t.Seconds() * s.BaseGBps * 1e9
}

// watermark 在风暴期从稳态 32 KB 线性冲到 200 KB，最后 20% 时间停在峰值上。
//
// 留一段平台是刻意的：真实队列越过 XOFF 后会贴着上限待一会儿，
// 而且没有平台的话，峰值存不存在完全取决于采样相位，对照实验没法讲清楚。
func (s *Signal) watermark(t time.Duration) uint64 {
	const steady = 32768
	const peak = 204800
	if !s.inStorm(t) {
		return steady
	}
	rem := t % s.StormPeriod
	frac := float64(rem-s.StormStart) / float64(s.StormLen)
	if frac > 0.8 {
		return peak
	}
	return uint64(float64(steady) + frac/0.8*float64(peak-steady))
}

func (s *Signal) operStatus(t time.Duration) string {
	if s.FlapPeriod <= 0 {
		return "UP"
	}
	rem := t % s.FlapPeriod
	if rem >= s.FlapAt && rem < s.FlapAt+s.FlapLen {
		return "DOWN"
	}
	return "UP"
}

// accumulate 计算周期事件在 [0, t] 内的累积量：rate × 总持续时间。
//
// 事件每个 period 出现一次，从 start 开始，持续 length。
// 用解析式而不是累加，保证真值精确、与采样间隔无关。
func accumulate(t, period, start, length time.Duration, rate float64) float64 {
	if t <= 0 || rate == 0 || period <= 0 || length <= 0 {
		return 0
	}
	perCycle := length.Seconds()
	full := t / period
	rem := t % period

	total := float64(full) * perCycle * rate
	switch {
	case rem >= start+length:
		total += perCycle * rate
	case rem > start:
		total += (rem - start).Seconds() * rate
	}
	return total
}

// rateBetween 由两个累积计数样本反推速率（每秒），这是设备遥测的通用口径。
func rateBetween(c0, c1 uint64, d time.Duration) float64 {
	if d <= 0 || c1 < c0 {
		return 0
	}
	return float64(c1-c0) / d.Seconds()
}
