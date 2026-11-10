package main

import (
	"fmt"
	"io"
	"sort"
)

// Report 打印整份实验报告。
func Report(w io.Writer, days int, samples []Sample, results []Result, timeB, eventB Budget, burns []BurnRow, fastPage bool, slo float64) {
	var totalReqs, totalBad int
	for _, s := range samples {
		totalReqs += s.Reqs
		totalBad += s.Bad
	}

	fmt.Fprintln(w, "== 09-slo · 三种 SLO 口径对照 ==")
	fmt.Fprintf(w, "窗口 %d 天 · %d 个一分钟窗口 · 请求 %d 条 · 失败 %d 条 · 目标 %.3f%%\n",
		days, len(samples), totalReqs, totalBad, slo*100)
	fmt.Fprintf(w, "判坏规则：失败率 > %.0f%% 或 P95 > %.0f ms\n\n", BadMinuteRate*100, BadLatencyMs)

	// 一、三种口径
	fmt.Fprintln(w, "一、同一份数据，三种口径")
	fmt.Fprintln(w, "------------------------------------------------------------")
	fmt.Fprintf(w, "  %-10s %10s %8s %-8s %s\n", "口径", "可用率", "差目标", "判定", "分母口径说明")
	for _, r := range results {
		fmt.Fprintf(w, "  %-10s %9.3f%% %7.1fbp %-8s %s\n",
			r.Caliber, r.Availability*100, r.GapBp(slo), r.Verdict(), r.Note)
	}
	fmt.Fprintf(w, "  %s\n\n", Compare(results))

	// 二、错误预算
	fmt.Fprintf(w, "二、错误预算（SLO %.3f%%）\n", slo*100)
	fmt.Fprintln(w, "------------------------------------------------------------")
	fmt.Fprintf(w, "  %-10s %12s %12s %12s %8s\n", "口径", "允许", "已消耗", "剩余", "已用")
	fmt.Fprintf(w, "  %-10s %10.1f 分 %10.1f 分 %10.1f 分 %7.1f%%\n",
		timeB.Caliber, timeB.Allowed, timeB.Consumed, timeB.Remaining, timeB.UsedPercent)
	fmt.Fprintf(w, "  %-10s %10.0f 条 %10.0f 条 %10.0f 条 %7.1f%%\n",
		eventB.Caliber, eventB.Allowed, eventB.Consumed, eventB.Remaining, eventB.UsedPercent)
	if eventB.Exhausted() {
		fmt.Fprintln(w, "  事件口径预算已耗尽：按这个速度，月底必然违约")
	} else {
		fmt.Fprintln(w, "  事件口径预算未耗尽：窗口还剩 " + fmt.Sprintf("%.1f%%", 100-eventB.UsedPercent) + " 的余量")
	}
	fmt.Fprintln(w, "")

	// 三、多窗口燃烧率
	fmt.Fprintln(w, "三、多窗口燃烧率（最近窗口）")
	fmt.Fprintln(w, "------------------------------------------------------------")
	fmt.Fprintf(w, "  %-8s %10s %10s %s\n", "窗口", "燃烧率", "阈值", "状态")
	for _, b := range burns {
		state := "正常"
		if b.Firing {
			state = "超阈值"
		}
		fmt.Fprintf(w, "  %-8s %9.1f× %9.1f× %s\n", fmt.Sprintf("%dm", b.WindowMin), b.Burn, b.Threshold, state)
	}
	if fastPage {
		fmt.Fprintln(w, "  判定：快烧两窗口同时超阈值 → page（人现在就该看）")
	} else {
		fmt.Fprintln(w, "  判定：未触发 page")
	}
	fmt.Fprintln(w, "")

	// 四、事故清单
	fmt.Fprintln(w, "四、注入的事故与三种口径的敏感度")
	fmt.Fprintln(w, "------------------------------------------------------------")
	fmt.Fprintf(w, "  %-24s %-12s %8s %10s %10s\n", "事故", "时段", "持续", "坏分钟", "坏请求")
	for _, in := range Incidents {
		bads, badReqs := windowImpact(samples, in.FromMin, in.ToMin)
		fmt.Fprintf(w, "  %-24s %-12s %7d 分 %10d %10d\n",
			in.Note, fmt.Sprintf("d%d %02d:%02d", in.FromMin/MinutesPerDay+1, (in.FromMin%MinutesPerDay)/60, in.FromMin%60),
			in.ToMin-in.FromMin, bads, badReqs)
	}
}

// windowImpact 返回窗口内的坏分钟数与坏请求数。
func windowImpact(samples []Sample, from, to int) (int, int) {
	badMin, badReqs := 0, 0
	for _, s := range samples {
		if s.TsMin < from || s.TsMin >= to {
			continue
		}
		if s.BadMinute() {
			badMin++
		}
		badReqs += s.Bad
	}
	return badMin, badReqs
}

// TopMinutes 返回失败最严重的 n 个分钟，报告用不到但测试与排障会用。
func TopMinutes(samples []Sample, n int) []Sample {
	cp := make([]Sample, len(samples))
	copy(cp, samples)
	sort.Slice(cp, func(i, j int) bool { return cp[i].FailRate() > cp[j].FailRate() })
	if n > len(cp) {
		n = len(cp)
	}
	return cp[:n]
}
