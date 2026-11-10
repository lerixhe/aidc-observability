package main

import (
	"fmt"
	"io"
)

// Report 打印整份实验报告。
func Report(w io.Writer, pts []Point, series []Series, ing IngestResult, windowFrom, windowTo int) {
	fmt.Fprintln(w, "== 08-pipeline · 秒级遥测落盘与查询保真度 ==")
	fmt.Fprintf(w, "窗口 %d s · 采样 %d s · 共 %d 点 · 阈值 %.0f KB · 劣化窗口 t=%d..%d\n\n",
		len(pts), SampleInterval, len(pts), QueueThresholdKB, DangerFromSec, DangerToSec)

	// 一、档位与体积
	fmt.Fprintln(w, "一、落盘档位与体积")
	fmt.Fprintln(w, "------------------------------------------------------------")
	for _, s := range series {
		fmt.Fprintf(w, "  %s\n", s.Describe())
	}
	raw := series[0].BucketBytes()
	minute := series[1].BucketBytes()
	if raw > 0 {
		fmt.Fprintf(w, "  压缩比：1m(max) 压到 1/%.0f，1d(max) 压到 1/%d\n\n",
			float64(raw)/float64(minute), int(float64(raw)/float64(maxInt(series[3].BucketBytes(), 1))))
	}

	// 二、窗口查询对照
	fmt.Fprintf(w, "二、事故窗口查询（t=%d..%d，共 %d s）\n", windowFrom, windowTo, windowTo-windowFrom)
	fmt.Fprintln(w, "------------------------------------------------------------")
	fmt.Fprintf(w, "  %-10s %6s %6s %9s %8s %8s %-6s %s\n",
		"档位", "桶数", "点数", "峰值KB", "P99KB", "均值KB", "告警", "短板")
	for _, s := range series {
		a := QueryWindow(s, windowFrom, windowTo)
		alarm := "否"
		if a.Alarmed {
			alarm = "是"
		}
		fmt.Fprintf(w, "  %-10s %6d %6d %9.1f %8.1f %8.1f %-6s %s\n",
			a.Tier, a.Buckets, a.Points, a.PeakKB, a.P99KB, a.MeanKB, alarm, a.TierNote())
	}
	fmt.Fprintln(w, "")

	// 三、背压与静默丢弃
	fmt.Fprintln(w, "三、采集器背压与静默丢弃")
	fmt.Fprintln(w, "------------------------------------------------------------")
	fmt.Fprintf(w, "  缓冲容量 %d 点 · 消费速率 %g 降档到 %g（t=%d..%d 期间）\n",
		ing.BufferCap, NormalRate, SlowRate, SlowStartSec, SlowEndSec)
	fmt.Fprintf(w, "  峰值缓冲占用 %d 点\n", ing.PeakBuffer)
	fmt.Fprintf(w, "  投递 %d / 期望 %d · 整体完整率 %.1f%%\n", len(ing.Delivered), ing.Expected, ing.Completeness()*100)
	fmt.Fprintf(w, "  丢弃 %d 点", ing.Dropped)
	if ing.FirstDropAt >= 0 {
		fmt.Fprintf(w, " · 首次丢弃 t=%d s", ing.FirstDropAt)
	}
	fmt.Fprintln(w, "")
	fmt.Fprintf(w, "  事故窗口（t=%d..%d）完整率 %.1f%% · 窗口内丢弃 %d 点\n",
		DangerFromSec, DangerToSec,
		ing.WindowCompleteness(DangerFromSec, DangerToSec)*100,
		ing.DroppedInWindow(DangerFromSec, DangerToSec))
	fmt.Fprintf(w, "  健康窗口（t=0..%d）完整率 %.1f%%\n", SlowStartSec,
		ing.WindowCompleteness(0, SlowStartSec)*100)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
