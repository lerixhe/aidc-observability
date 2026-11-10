package main

import (
	"fmt"
	"io"
	"math"
	"net/http/httptest"
	"strings"
	"time"
)

// 本 lab 的路径常量，bench 与报告共用。
const (
	pathPause      = "/interfaces/interface[name=Ethernet0]/state/counters/out-pfc-pause-frames"
	pathWatermark  = "/qos/interfaces/interface[id=Ethernet0]/output/queues/queue[name=3]/state/watermark-bytes"
	pathOperStatus = "/interfaces/interface[name=Ethernet0]/state/oper-status"
)

var benchIntervals = []time.Duration{
	100 * time.Millisecond,
	time.Second,
	5 * time.Second,
	10 * time.Second,
}

// Bench 是一次保真度对照的完整结果。
type Bench struct {
	Speed       float64
	Duration    time.Duration
	TruePausePk float64 // 真实峰值速率，帧/秒
	TrueWM      float64 // 真实水位峰值，字节
	TrueDowns   int     // 真实 DOWN 次数

	Counter []CounterRow
	Gauge   []GaugeRow
	State   []StateRow
}

// CounterRow 是一组「累积计数器」的采样结果。
type CounterRow struct {
	Mode         string
	Interval     time.Duration
	Samples      int
	PeakRate     float64
	Fidelity     float64 // 观测峰值 / 真实峰值
	Bytes        int
	PointsPerSec float64
}

// GaugeRow 是一组「瞬时水位」的采样结果。
type GaugeRow struct {
	Mode      string
	Interval  time.Duration
	Samples   int
	PeakValue float64
	Fidelity  float64
	Bytes     int
}

// StateRow 是一组「离散状态」的采样结果。
type StateRow struct {
	Mode     string
	Interval time.Duration
	Samples  int
	Downs    int
	Recall   float64 // 检出的 DOWN 次数 / 真实次数
	Bytes    int
}

// RunBench 起一个真实 HTTP 服务，用不同订阅策略把同一份信号采一遍。
func RunBench(duration time.Duration, speed float64) (*Bench, error) {
	sig := DefaultSignal()
	// httptest 起的是真实 HTTP 服务，只是端口不固定；订阅走的就是这条真链路
	srv := httptest.NewUnstartedServer(NewServer(sig, speed).Handler())
	srv.Start()
	defer srv.Close()
	base := srv.URL

	b := &Bench{
		Speed:       speed,
		Duration:    duration,
		TruePausePk: sig.TruePeakPauseRate(),
		TrueWM:      204800,
		TrueDowns:   trueDownCount(sig, duration),
	}

	modes := []struct {
		name string
		mode string
	}{
		{"SAMPLE", "sample"},
		{"ON_CHANGE", "on_change"},
	}

	for _, m := range modes {
		for _, iv := range benchIntervals {
			sub := subscribeFast(base, pathPause, m.mode, iv, duration)
			if sub.Err != nil {
				return nil, fmt.Errorf("%s %s 订阅失败：%w", m.name, iv, sub.Err)
			}
			samples := sub.series(pathPause)
			peak := peakRate(samples)
			b.Counter = append(b.Counter, CounterRow{
				Mode:         m.name,
				Interval:     iv,
				Samples:      len(samples),
				PeakRate:     peak,
				Fidelity:     math.Min(1, safeDiv(peak, b.TruePausePk)),
				Bytes:        sub.Bytes,
				PointsPerSec: safeDiv(float64(len(samples)), duration.Seconds()),
			})
		}
	}

	for _, m := range modes {
		for _, iv := range benchIntervals {
			sub := subscribeFast(base, pathWatermark, m.mode, iv, duration)
			if sub.Err != nil {
				return nil, fmt.Errorf("水位 %s %s 订阅失败：%w", m.name, iv, sub.Err)
			}
			samples := sub.series(pathWatermark)
			peak := peakValue(samples)
			b.Gauge = append(b.Gauge, GaugeRow{
				Mode:      m.name,
				Interval:  iv,
				Samples:   len(samples),
				PeakValue: peak,
				Fidelity:  math.Min(1, safeDiv(peak, b.TrueWM)),
				Bytes:     sub.Bytes,
			})
		}
	}

	for _, m := range modes {
		for _, iv := range benchIntervals {
			sub := subscribeFast(base, pathOperStatus, m.mode, iv, duration)
			if sub.Err != nil {
				return nil, fmt.Errorf("状态 %s %s 订阅失败：%w", m.name, iv, sub.Err)
			}
			samples := sub.series(pathOperStatus)
			downs := countState(samples, "DOWN")
			b.State = append(b.State, StateRow{
				Mode:     m.name,
				Interval: iv,
				Samples:  len(samples),
				Downs:    downs,
				Recall:   safeDiv(float64(downs), float64(b.TrueDowns)),
				Bytes:    sub.Bytes,
			})
		}
	}

	return b, nil
}

// trueDownCount 用 10 ms 步进统计真实的 DOWN 次数。
func trueDownCount(sig *Signal, duration time.Duration) int {
	n := 0
	prev := ""
	for t := time.Duration(0); t < duration; t += 10 * time.Millisecond {
		s := sig.operStatus(t)
		if s == "DOWN" && prev != "DOWN" {
			n++
		}
		prev = s
	}
	return n
}

// Report 打印三张保真度表。
func Report(w io.Writer, b *Bench) {
	fmt.Fprintf(w, "== 06-gnmi-mock · 订阅策略保真度对照 ==\n")
	fmt.Fprintf(w, "虚拟时长 %.0f s · 加速 %.0f×（真实耗时约 %.1f s）\n",
		b.Duration.Seconds(), b.Speed, b.Duration.Seconds()/b.Speed)
	fmt.Fprintf(w, "信号：PFC 风暴每 20 s 一次、每次 1.5 s、速率 %.0f 帧/s；水位从 32 KB 冲到 %.0f KB\n\n",
		b.TruePausePk, b.TrueWM/1024)

	fmt.Fprintln(w, "① 累积计数器：out-pfc-pause-frames")
	fmt.Fprintf(w, "%-10s %8s %8s %14s %8s %10s\n", "模式", "间隔", "样本数", "观测峰值帧/s", "保真度", "数据量")
	fmt.Println(strings.Repeat("-", 64))
	for _, r := range b.Counter {
		fmt.Fprintf(w, "%-10s %8s %8d %14.0f %7.0f%% %10s\n",
			r.Mode, r.Interval, r.Samples, r.PeakRate, r.Fidelity*100, humanBytes(r.Bytes))
	}
	fmt.Fprintf(w, "\n真实峰值 %.0f 帧/s。判定标准：观测峰值 / 真实峰值。\n", b.TruePausePk)

	fmt.Fprintln(w, "\n② 瞬时水位：queue[3] watermark-bytes")
	fmt.Fprintf(w, "%-10s %8s %8s %14s %8s %10s\n", "模式", "间隔", "样本数", "观测峰值KB", "保真度", "数据量")
	fmt.Println(strings.Repeat("-", 64))
	for _, r := range b.Gauge {
		fmt.Fprintf(w, "%-10s %8s %8d %14.0f %7.0f%% %10s\n",
			r.Mode, r.Interval, r.Samples, r.PeakValue/1024, r.Fidelity*100, humanBytes(r.Bytes))
	}

	fmt.Fprintln(w, "\n③ 离散状态：oper-status")
	fmt.Fprintf(w, "%-10s %8s %8s %10s %8s %10s\n", "模式", "间隔", "样本数", "DOWN次数", "召回率", "数据量")
	fmt.Println(strings.Repeat("-", 64))
	for _, r := range b.State {
		fmt.Fprintf(w, "%-10s %8s %8d %10d %7.0f%% %10s\n",
			r.Mode, r.Interval, r.Samples, r.Downs, r.Recall*100, humanBytes(r.Bytes))
	}
	fmt.Fprintf(w, "\n真实 DOWN 次数 %d 次（每次 300 ms，每 7 s 一次）。\n", b.TrueDowns)
}

func humanBytes(n int) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%d B", n)
}

func safeDiv(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}
