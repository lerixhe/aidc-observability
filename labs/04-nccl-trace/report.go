package main

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// Report 打印从日志还原出的流量矩阵与核算结论。
func Report(w io.Writer, l *Log, a *Analysis, path string, portGBps float64) {
	fmt.Fprintf(w, "== 04-nccl-trace · 从日志还原流量矩阵 ==\n")
	fmt.Fprintf(w, "日志 %s · 识别 %d 行，忽略 %d 行", path, l.ParsedLines, l.IgnoredLines)
	if l.ANSIStripped > 0 {
		fmt.Fprintf(w, "（剥掉 %d 行 ANSI 转义）", l.ANSIStripped)
	}
	fmt.Fprintln(w)
	fmt.Fprintf(w, "任务：%d rank · allReduce · 每步消息 %.1f MB · %d 步\n",
		a.N, a.MsgBytes/(1<<20), len(a.Steps))

	src := "来自日志 Ring 行"
	if !a.RingFromLog {
		src = "日志无 Ring 行，已从发送关系反推"
	}
	fmt.Fprintf(w, "环：%s -> %d（%s）\n\n", joinRing(a.Ring), a.Ring[0], src)

	fmt.Fprintf(w, "逐步核算（同步通信，步耗时 = 最慢 rank）\n")
	fmt.Fprintf(w, "%-6s %10s %8s %8s %8s  %s\n", "step", "耗时ms", "algbw", "busbw", "最慢rank", "判定")
	fmt.Println(repeatChar('-', 62))
	for _, s := range a.Steps {
		verdict := "正常"
		if s.Anomaly {
			verdict = fmt.Sprintf("慢 %.1f×", s.BusyUS/a.BaselineUS)
		}
		fmt.Fprintf(w, "%-6d %10.2f %8.1f %8.1f %8d  %s\n",
			s.Step, s.BusyUS/1000, s.AlgBW, s.BusBW, s.SlowRank, verdict)
	}
	fmt.Fprintln(w)

	fmt.Fprintf(w, "基线 busbw %.1f GB/s（端口 %.0f GB/s，效率 %.0f%%）\n",
		a.Baseline, portGBps, a.Efficiency(portGBps)*100)
	if len(a.AnomalySteps) > 0 {
		fmt.Fprintf(w, "异常步 %s · 最慢 rank 集中在 %d（%d 步里 %d 步）\n",
			joinInts(a.AnomalySteps), a.SuspectRank, len(a.AnomalySteps), countRank(a, a.SuspectRank))
		fmt.Fprintf(w, "多花 %.0f 卡·毫秒，全部是等待时间（同步通信里，等最慢的等于全员一起等）\n", a.WastedRankUS()/1e3)
	} else {
		fmt.Fprintln(w, "没有检测到异常步")
	}
	// 基线污染检查：故障步过半时，中位数本身就会被拉低，
	// 于是「大家都是慢的」看起来一切正常。这时用 P75/P50 的比值报警。
	p25, p75 := quartiles(a)
	if p25 > 0 && p75/p25 > 1.5 {
		fmt.Fprintf(w, "⚠ 步耗时呈二峰分布（P25 %.1f / P75 %.1f GB/s）：慢步可能占了多数，\n"+
			"  中位数基线已被污染。此时应改用 P75 或历史基线做参照。\n", p25, p75)
	}
	fmt.Fprintln(w)

	fmt.Fprintf(w, "流量矩阵（每步每个 rank 发给下一跳的 MB，行 = 源，列 = 目的）\n")
	printMatrix(w, a)

	fmt.Fprintln(w)
	fmt.Fprintln(w, "预期 vs 实际")
	fmt.Fprintf(w, "  预期：每步 %.2f ms（%.1f MB ÷ %.1f GB/s × 2(N-1)/N）\n",
		a.BaselineUS/1000, a.MsgBytes/(1<<20), a.Baseline)
	fmt.Fprintf(w, "  实际：%.2f ms 到 %.2f ms 之间，%d 步超出预期 25%% 以上\n",
		minUS(a)/1000, maxUS(a)/1000, len(a.AnomalySteps))
	fmt.Fprintln(w, "  账本在任务元数据里：环怎么排、每步发多少，都是已知量。")
	fmt.Fprintln(w, "  偏差本身就是信号，不用等训练侧报慢。")
}

func printMatrix(w io.Writer, a *Analysis) {
	n := a.N
	fmt.Fprintf(w, "%6s", "src\\dst")
	for d := 0; d < n; d++ {
		fmt.Fprintf(w, "%8d", d)
	}
	fmt.Fprintln(w)
	for r := 0; r < n; r++ {
		fmt.Fprintf(w, "%6d", r)
		for d := 0; d < n; d++ {
			v := a.Matrix[[2]int{r, d}]
			if v == 0 {
				fmt.Fprintf(w, "%8s", "·")
				continue
			}
			fmt.Fprintf(w, "%8.1f", v/(1<<20))
		}
		fmt.Fprintln(w)
	}
}

// quartiles 返回 busbw 的四分位，用于判断分布是否二峰。
func quartiles(a *Analysis) (float64, float64) {
	vals := make([]float64, len(a.Steps))
	for i, s := range a.Steps {
		vals[i] = s.BusBW
	}
	sort.Float64s(vals)
	if len(vals) < 4 {
		return 0, 0
	}
	return vals[len(vals)/4], vals[len(vals)*3/4]
}

func countRank(a *Analysis, rank int) int {
	c := 0
	for _, s := range a.Steps {
		if s.Anomaly && s.SlowRank == rank {
			c++
		}
	}
	return c
}

func minUS(a *Analysis) float64 {
	v := a.Steps[0].BusyUS
	for _, s := range a.Steps {
		if s.BusyUS < v {
			v = s.BusyUS
		}
	}
	return v
}

func maxUS(a *Analysis) float64 {
	v := a.Steps[0].BusyUS
	for _, s := range a.Steps {
		if s.BusyUS > v {
			v = s.BusyUS
		}
	}
	return v
}

func joinInts(vals []int) string {
	sort.Ints(vals)
	parts := make([]string, len(vals))
	for i, v := range vals {
		parts[i] = fmt.Sprintf("%d", v)
	}
	return strings.Join(parts, ",")
}

func repeatChar(c rune, n int) string {
	out := make([]rune, n)
	for i := range out {
		out[i] = c
	}
	return string(out)
}
