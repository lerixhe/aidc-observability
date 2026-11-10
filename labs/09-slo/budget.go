package main

// Budget 是一种口径下的错误预算。
type Budget struct {
	Caliber     Caliber
	Allowed     float64 // 允许的坏单位数（分钟或请求数）
	Consumed    float64 // 已消耗
	Remaining   float64
	UsedPercent float64
}

// TimeBudget 算时间口径的错误预算。
func TimeBudget(samples []Sample, slo float64) Budget {
	allowed := float64(len(samples)) * (1 - slo)
	consumed := 0.0
	for _, s := range samples {
		if s.BadMinute() {
			consumed++
		}
	}
	return newBudget(CaliberTime, allowed, consumed)
}

// EventBudget 算事件口径的错误预算。
func EventBudget(samples []Sample, slo float64) Budget {
	var allowed, consumed float64
	var totalReqs int
	for _, s := range samples {
		totalReqs += s.Reqs
		consumed += float64(s.Bad)
	}
	allowed = float64(totalReqs) * (1 - slo)
	return newBudget(CaliberEvent, allowed, consumed)
}

func newBudget(c Caliber, allowed, consumed float64) Budget {
	b := Budget{Caliber: c, Allowed: allowed, Consumed: consumed}
	b.Remaining = allowed - consumed
	if allowed > 0 {
		b.UsedPercent = consumed / allowed * 100
	}
	return b
}

// Exhausted 表示预算已经用光。
func (b Budget) Exhausted() bool { return b.Remaining <= 0 }

// BurnRow 是一个时间窗口上的燃烧率。
type BurnRow struct {
	WindowMin int
	Burn      float64
	Threshold float64
	Firing    bool
}

// 多窗口燃烧率的参考阈值，来自 SRE Workbook 的推荐配置。
const (
	BurnFastWindow = 5
	BurnFastLong   = 60
	BurnFastThresh = 14.4
	BurnSlowWindow = 30
	BurnSlowLong   = 360
	BurnSlowThresh = 6.0
)

// Tail 取最后 n 分钟的样本。
func Tail(samples []Sample, minutes int) []Sample {
	if minutes <= 0 || minutes >= len(samples) {
		return samples
	}
	return samples[len(samples)-minutes:]
}

// BurnRate 算最近 windowMin 分钟的燃烧率：
//
//	burn = (1 - 窗口可用率) / (1 - SLO)
//
// 1 表示「按这个速度刚好用完预算」，14.4 表示按这个速度 1 小时烧掉 2% 的月度预算。
func BurnRate(samples []Sample, windowMin int, slo float64) float64 {
	w := Tail(samples, windowMin)
	if len(w) == 0 {
		return 0
	}
	r := Compute(CaliberEvent, w, slo)
	if r.Availability >= 1 {
		return 0
	}
	return (1 - r.Availability) / (1 - slo)
}

// MultiWindowBurn 返回四个窗口的燃烧率，并给出两对是否同时触发。
func MultiWindowBurn(samples []Sample, slo float64) ([]BurnRow, bool) {
	rows := []BurnRow{
		{WindowMin: BurnFastWindow, Threshold: BurnFastThresh},
		{WindowMin: BurnFastLong, Threshold: BurnFastThresh},
		{WindowMin: BurnSlowWindow, Threshold: BurnSlowThresh},
		{WindowMin: BurnSlowLong, Threshold: BurnSlowThresh},
	}
	for i := range rows {
		rows[i].Burn = BurnRate(samples, rows[i].WindowMin, slo)
		rows[i].Firing = rows[i].Burn >= rows[i].Threshold
	}
	// 快烧：5m 与 1h 同时超阈值 → page
	fast := rows[0].Firing && rows[1].Firing
	return rows, fast
}
