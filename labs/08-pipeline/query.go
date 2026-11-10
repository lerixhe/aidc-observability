package main

// WindowAnswer 是对一个时间窗口的查询结果。
type WindowAnswer struct {
	Tier    Tier
	Buckets int   // 命中的桶数
	Points  int   // 命中的原始点数（仅有原始档才准确）
	PeakKB  float64
	P99KB   float64
	MeanKB  float64
	// Alarmed 表示按阈值判断是否会告警
	Alarmed bool
	// TimeRes 是这一档能定位到的时间分辨率（秒）
	TimeRes int
}

// QueryWindow 在某一档数据上查询窗口。
//
// 关键点：只有保留了峰值的档位才能回答「峰值多少」。
// 只留均值的档位，PeakKB 与 P99KB 都退化成均值。
func QueryWindow(s Series, fromSec, toSec int) WindowAnswer {
	a := WindowAnswer{Tier: s.Tier, TimeRes: s.PointsPerBucket}
	var maxV, sumV float64
	var p99s []float64
	for _, b := range s.Buckets {
		if b.ToSec <= fromSec || b.FromSec >= toSec {
			continue
		}
		a.Buckets++
		a.Points += b.Count
		sumV += b.Mean * float64(b.Count)
		if b.Max > maxV {
			maxV = b.Max
		}
		p99s = append(p99s, b.P99)
	}
	if a.Buckets > 0 && a.Points > 0 {
		a.MeanKB = round1(sumV / float64(a.Points))
	}
	if maxV == 0 && len(p99s) > 0 {
		// 只留均值的档位：峰值信息不存在，用均值近似（这正是坑所在）
		maxV = a.MeanKB
	}
	a.PeakKB = maxV

	// P99 的算法分两种口径：
	//   原始档（每桶 1 点）：窗口内全部点重算，准确。
	//   聚合档：只能取各桶 P99 的最大值，当作下界（保守）。
	if a.TimeRes == 1 {
		vals := make([]float64, 0, len(p99s))
		vals = append(vals, p99s...)
		a.P99KB = round1(quantile(vals, 0.99))
	} else {
		var p float64
		for _, v := range p99s {
			if v > p {
				p = v
			}
		}
		a.P99KB = p
	}
	if a.P99KB == 0 {
		a.P99KB = a.MeanKB
	}
	a.Alarmed = a.PeakKB >= QueueThresholdKB
	return a
}

// TierNote 给出这一档在本次查询里的短板说明。
func (a WindowAnswer) TierNote() string {
	switch {
	case a.Tier == TierRaw:
		return "完整信息，可定位到秒"
	case a.Tier == TierMinuteMax:
		return "峰值保住，时间分辨率降到分钟"
	case a.Tier == TierMinuteMean:
		return "峰值丢失，短时冲顶被摊平"
	case a.Tier == TierDailyMax:
		return "峰值保住，但无法定位到哪一分钟"
	}
	return ""
}
