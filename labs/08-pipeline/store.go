package main

import (
	"fmt"
	"math"
	"sort"
)

// Tier 是落盘档位。四档的差别就是课 14 讲的那个选择：留均值还是留峰值。
type Tier string

const (
	// TierRaw 每个秒级点都留。
	TierRaw Tier = "raw"
	// TierMinuteMax 分钟聚合，均值 + 峰值 + P99。
	TierMinuteMax Tier = "1m(max)"
	// TierMinuteMean 分钟聚合，只留均值。这是最常见的错误配置。
	TierMinuteMean Tier = "1m(mean)"
	// TierDailyMax 日聚合，均值 + 峰值 + P99。
	TierDailyMax Tier = "1d(max)"
)

// Tiers 是报告里的展示顺序。
var Tiers = []Tier{TierRaw, TierMinuteMax, TierMinuteMean, TierDailyMax}

// Bucket 是一个聚合桶。
type Bucket struct {
	FromSec int
	ToSec   int
	Count   int
	Mean    float64
	Max     float64
	P99     float64
}

// Bytes 返回这个桶序列的理论落盘字节数（粗估，每值 8 字节 + 桶头 32 字节）。
func (b Bucket) Bytes() int { return 32 + 3*8 }

// Series 是一档数据的完整形态。
type Series struct {
	Tier    Tier
	Buckets []Bucket
	// PointsPerBucket 说明这一档把多少个秒级点压成一个桶
	PointsPerBucket int
}

// BucketBytes 返回整档的落盘字节数。
func (s Series) BucketBytes() int {
	n := 0
	for _, b := range s.Buckets {
		n += b.Bytes()
	}
	return n
}

// Aggregate 把一个窗口内的点聚合成一个桶。
func Aggregate(pts []Point, fromSec, toSec int, keepPeak bool) Bucket {
	vals := make([]float64, 0, len(pts))
	sum := 0.0
	for _, p := range pts {
		if p.TsSec < fromSec || p.TsSec >= toSec {
			continue
		}
		vals = append(vals, p.KB)
		sum += p.KB
	}
	b := Bucket{FromSec: fromSec, ToSec: toSec, Count: len(vals)}
	if len(vals) == 0 {
		return b
	}
	b.Mean = round1(sum / float64(len(vals)))
	if keepPeak {
		b.Max = round1(maxOf(vals))
		b.P99 = round1(quantile(vals, 0.99))
	}
	return b
}

// BuildSeries 按档位切桶聚合。
func BuildSeries(tier Tier, pts []Point) Series {
	switch tier {
	case TierRaw:
		buckets := make([]Bucket, 0, len(pts))
		for _, p := range pts {
			buckets = append(buckets, Bucket{
				FromSec: p.TsSec, ToSec: p.TsSec + 1, Count: 1,
				Mean: p.KB, Max: p.KB, P99: p.KB,
			})
		}
		return Series{Tier: tier, Buckets: buckets, PointsPerBucket: 1}
	case TierMinuteMax:
		return build(tier, pts, 60, true)
	case TierMinuteMean:
		return build(tier, pts, 60, false)
	case TierDailyMax:
		return build(tier, pts, 86400, true)
	}
	return Series{Tier: tier}
}

func build(tier Tier, pts []Point, width int, keepPeak bool) Series {
	if len(pts) == 0 {
		return Series{Tier: tier, PointsPerBucket: width}
	}
	from := pts[0].TsSec
	to := pts[len(pts)-1].TsSec + 1
	var buckets []Bucket
	for s := from; s < to; s += width {
		end := s + width
		b := Aggregate(pts, s, end, keepPeak)
		if b.Count == 0 {
			continue
		}
		buckets = append(buckets, b)
	}
	return Series{Tier: tier, Buckets: buckets, PointsPerBucket: width}
}

// maxOf 返回最大值。
func maxOf(vals []float64) float64 {
	m := vals[0]
	for _, v := range vals[1:] {
		if v > m {
			m = v
		}
	}
	return m
}

// quantile 用最近秩法算分位数：idx = ceil(q*n) - 1。
//
// 窗口内点数不多，直接排序，避免直方图误差。
// 课 14 强调分位数不能跨实例平均，这里保留的是「窗口内重算」，所以准确。
func quantile(vals []float64, q float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	cp := make([]float64, len(vals))
	copy(cp, vals)
	sort.Float64s(cp)
	idx := int(math.Ceil(q*float64(len(cp)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(cp) {
		idx = len(cp) - 1
	}
	return cp[idx]
}

// Describe 给报告用的一行说明。
func (s Series) Describe() string {
	return fmt.Sprintf("%-10s 桶 %4d 个 · 每桶 %5d 点 · 落盘 %7d 字节",
		s.Tier, len(s.Buckets), s.PointsPerBucket, s.BucketBytes())
}
