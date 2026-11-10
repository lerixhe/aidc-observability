package main

import "sort"

// Summary 是一段流量序列的统计画像。
type Summary struct {
	Name      string
	Mean      float64
	P50       float64
	P90       float64
	P99       float64
	P999      float64
	Max       float64
	PeakRatio float64 // 峰值 / 均值，同步突发的放大倍数
	ZeroRatio float64 // 空闲时间占比
}

// Summarize 算分位与放大倍数。
func Summarize(t Traffic) Summary {
	sorted := append([]float64(nil), t.GBps...)
	sort.Float64s(sorted)
	mean := t.Mean()
	zeros := 0
	for _, v := range t.GBps {
		if v == 0 {
			zeros++
		}
	}
	n := len(sorted)
	s := Summary{
		Name: t.Name,
		Mean: mean,
		P50:  quantile(sorted, 0.50),
		P90:  quantile(sorted, 0.90),
		P99:  quantile(sorted, 0.99),
		P999: quantile(sorted, 0.999),
		Max:  sorted[n-1],
	}
	if mean > 0 {
		s.PeakRatio = s.Max / mean
	}
	if n > 0 {
		s.ZeroRatio = float64(zeros) / float64(n)
	}
	return s
}

func quantile(sorted []float64, q float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(q * float64(len(sorted)-1))
	return sorted[idx]
}

// PolledPeak 模拟「轮询采样」：用 windowMS 的窗口做滑动平均，
// 返回你在面板上能看到的最高值。窗口越长，突发被抹得越平。
func PolledPeak(vals []float64, windowMS int) float64 {
	if windowMS <= 1 {
		return maxOf(vals)
	}
	if windowMS > len(vals) {
		/// 窗口比观测时长还长，只能得到一个平均值
		return meanOf(vals)
	}
	// 前缀和，滑动窗口 O(n)
	prefix := make([]float64, len(vals)+1)
	for i, v := range vals {
		prefix[i+1] = prefix[i] + v
	}
	best := 0.0
	for i := windowMS; i <= len(vals); i++ {
		avg := (prefix[i] - prefix[i-windowMS]) / float64(windowMS)
		if avg > best {
			best = avg
		}
	}
	return best
}

func maxOf(vals []float64) float64 {
	hi := 0.0
	for _, v := range vals {
		if v > hi {
			hi = v
		}
	}
	return hi
}

func meanOf(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}
