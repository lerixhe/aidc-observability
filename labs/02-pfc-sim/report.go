package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// WriteCSV 把采样序列落成 CSV，方便读者用任意工具画图。
func WriteCSV(path string, samples []Sample) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := &strings.Builder{}
	fmt.Fprintln(w, "t_us,ingress_max_kb,egress_kb,paused_ports,ecn_marked_kb,dropped_kb,sent_kb")
	for _, s := range samples {
		fmt.Fprintf(w, "%.0f,%.1f,%.1f,%d,%.1f,%.1f,%.1f\n",
			s.T, s.IngressMaxKB, s.EgressKB, s.PausedPorts, s.ECNMarkedKB, s.DroppedKB, s.SentKB)
	}
	_, err = io.WriteString(f, w.String())
	return err
}

// Report 打印一轮仿真的摘要，格式固定，便于和 expected/summary.txt 对照。
func Report(w io.Writer, cfg Config, st Stats, samples []Sample) {
	pfc := "开"
	if !cfg.PFC {
		pfc = "关"
	}
	dcqcn := "关"
	if cfg.DCQCN {
		dcqcn = "开"
	}
	fmt.Fprintf(w, "== 02-pfc-sim · %s ==\n", modeTitle(cfg.Mode))
	fmt.Fprintf(w, "上游 %d 端口 × %.0f KB/μs（%.0f Gbps） · 出端口 %.0f KB/μs\n",
		cfg.Ports, cfg.PortRate, cfg.PortRate*8, cfg.PortRate)
	fmt.Fprintf(w, "PFC %s · XOFF %.0f%% / XON %.0f%% of %.0f KB · pause 延迟 %.0f μs · 配额 %.0f μs\n",
		pfc, cfg.XOFFRatio*100, cfg.XONRatio*100, cfg.IngressBuf, cfg.PauseDelay, cfg.PauseQuota)
	fmt.Fprintf(w, "ECN 水线 %.0f%%–%.0f%% of %.0f KB · DCQCN %s\n",
		cfg.KminRatio*100, cfg.KmaxRatio*100, cfg.EgressBuf, dcqcn)
	fmt.Fprintf(w, "时长 %.0f μs · 同步周期 %.0f μs · 突发占比 %.0f%%\n\n",
		cfg.Duration, cfg.Period, cfg.BurstRatio*100)

	util := safeDiv(st.SentKB, cfg.PortRate*cfg.Duration)
	fmt.Fprintf(w, "%-22s %12s\n", "指标", "值")
	row := func(k, v string) { fmt.Fprintf(w, "%-22s %12s\n", k, v) }
	row("注入流量", fmt.Sprintf("%.0f KB", st.InjectedKB))
	row("送出流量", fmt.Sprintf("%.0f KB", st.SentKB))
	row("出端口利用率", fmt.Sprintf("%.1f%%", util*100))
	row("溢出丢弃", fmt.Sprintf("%.0f KB", st.DroppedKB))
	row("ECN 标记", fmt.Sprintf("%.0f KB (%.1f%%)", st.ECNMarkedKB, safeDiv(st.ECNMarkedKB, st.SentKB)*100))
	row("PFC pause 帧", fmt.Sprintf("%d", st.PauseFrames))
	row("PFC XON 帧", fmt.Sprintf("%d", st.PauseResumes))
	row("反压覆盖率", fmt.Sprintf("%.1f%%", st.PauseCoverage(cfg.Ports, cfg.Duration)*100))
	row("最长连续暂停", fmt.Sprintf("%.0f μs", st.MaxPauseUS))
	row("watchdog 触发", fmt.Sprintf("%d", st.WatchdogHits))
	row("出端口水位峰值", fmt.Sprintf("%.0f KB", st.EgressPeak))
	row("出端口水位均值", fmt.Sprintf("%.0f KB", safeDiv(st.EgressSum, cfg.Duration)))
	row("ingress 水位峰值", fmt.Sprintf("%.0f KB", st.IngressPeak))

	if cfg.Mode == "hol" {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "队头阻塞对照（F1 = 去慢目的地的流，F2 = 去健康目的地的无辜流）\n")
		fmt.Fprintf(w, "  %-16s %12s %12s %10s\n", "流", "需求 KB", "送出 KB", "达成率")
		fmt.Fprintf(w, "  %-16s %12.0f %12.0f %9.1f%%\n", "F1 慢目的地", st.F1WantedKB, st.F1SentKB, safeDiv(st.F1SentKB, st.F1WantedKB)*100)
		fmt.Fprintf(w, "  %-16s %12.0f %12.0f %9.1f%%\n", "F2 无辜流", st.F2WantedKB, st.F2SentKB, safeDiv(st.F2SentKB, st.F2WantedKB)*100)
	}

	if len(samples) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "时序（每列取一个采样窗口的最大值）")
		printSpark(w, "出端口缓冲 KB ", pluck(samples, func(s Sample) float64 { return s.EgressKB }), "KB")
		printSpark(w, "PFC 暂停端口数", pluck(samples, func(s Sample) float64 { return float64(s.PausedPorts) }), "")
		printSpark(w, "ECN 标记 KB  ", pluck(samples, func(s Sample) float64 { return s.ECNMarkedKB }), "KB")
		printSpark(w, "溢出丢弃 KB  ", pluck(samples, func(s Sample) float64 { return s.DroppedKB }), "KB")
	}
}

func modeTitle(m string) string {
	switch m {
	case "incast":
		return "AllReduce 同步突发（incast）"
	case "idc":
		return "传统 IDC 小流（对照）"
	case "hol":
		return "队头阻塞（HoL）"
	}
	return m
}

func pluck(samples []Sample, f func(Sample) float64) []float64 {
	out := make([]float64, len(samples))
	for i, s := range samples {
		out[i] = f(s)
	}
	return out
}

var sparks = []rune("▁▂▃▄▅▆▇█")

// printSpark 画单行火花线：一眼看出形状，不用装画图库。
func printSpark(w io.Writer, label string, vals []float64, unit string) {
	if len(vals) == 0 {
		return
	}
	const width = 96
	// 降采样：每列取窗口内最大值，峰值不会被平均掉
	step := float64(len(vals)) / width
	if step < 1 {
		step = 1
	}
	cols := []float64{}
	for i := 0.0; i < float64(len(vals)); i += step {
		hi := 0.0
		for j := int(i); j < int(i+step) && j < len(vals); j++ {
			if vals[j] > hi {
				hi = vals[j]
			}
		}
		cols = append(cols, hi)
	}
	peak := 0.0
	sum := 0.0
	for _, v := range vals {
		if v > peak {
			peak = v
		}
		sum += v
	}
	if peak <= 0 {
		fmt.Fprintf(w, "%s 全程为 0\n", label)
		return
	}
	b := &strings.Builder{}
	for _, v := range cols {
		idx := int(v / peak * float64(len(sparks)-1))
		if v > 0 && idx == 0 {
			idx = 1 // 非零就别画成空格感
		}
		b.WriteRune(sparks[idx])
	}
	fmt.Fprintf(w, "%s 峰 %.0f%s 均 %.0f%s │%s│\n", label, peak, unit, sum/float64(len(vals)), unit, b.String())
}

func safeDiv(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}
