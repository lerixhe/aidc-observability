package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Report 打印对照结论。格式固定，便于和 expected/summary.txt 比对。
func Report(w io.Writer, p Params, idc, ar Traffic) {
	fmt.Fprintf(w, "== 01-traffic-model · 同平均负载，不同到达形态 ==\n")
	fmt.Fprintf(w, "观测 %d s · 目标平均负载 %.0f GB/s · 单端口线速 %.0f GB/s（200 Gbps）· %d worker 汇聚\n",
		p.DurationMS/1000, p.MeanGBps, p.PortGBps, p.Workers)
	fmt.Fprintln(w)

	si := Summarize(idc)
	sa := Summarize(ar)

	fmt.Fprintf(w, "%-22s %10s %10s\n", "统计量（GB/s）", si.Name, sa.Name)
	row := func(k string, a, b float64) {
		fmt.Fprintf(w, "%-22s %10.1f %10.1f\n", k, a, b)
	}
	row("平均负载", si.Mean, sa.Mean)
	row("P50", si.P50, sa.P50)
	row("P90", si.P90, sa.P90)
	row("P99", si.P99, sa.P99)
	row("峰值", si.Max, sa.Max)
	row("峰值 / 均值", si.PeakRatio, sa.PeakRatio)
	fmt.Fprintf(w, "%-22s %9.0f%% %9.0f%%\n", "空闲时间占比", si.ZeroRatio*100, sa.ZeroRatio*100)
	fmt.Fprintln(w)

	// 轮询失真：这才是本 lab 的核心结论
	fmt.Fprintf(w, "轮询失真（滑动平均后面板能看到的最高值，GB/s）\n")
	fmt.Fprintf(w, "%-10s %14s %18s %s\n", "采样窗口", si.Name, sa.Name, "对 AllReduce 的判断")
	fmt.Println(repeatChar('-', 78))
	for _, win := range []int{1, 10, 100, 1000, 10000} {
		a := PolledPeak(idc.GBps, win)
		b := PolledPeak(ar.GBps, win)
		verdict := "看得见突发"
		if b < p.PortGBps*0.5 {
			verdict = "被抹平，看起来一切正常"
		} else if b < p.PortGBps {
			verdict = "峰值被低估"
		}
		fmt.Fprintf(w, "%-10s %13.1f %17.1f   %s\n", winLabel(win), a, b, verdict)
	}
	fmt.Fprintln(w)

	fmt.Fprintln(w, "时序（只画前 2 s，1 ms 分辨率，每列取窗口最大值）")
	printSpark(w, si.Name, head(idc.GBps, 2000), "GB/s")
	printSpark(w, sa.Name, head(ar.GBps, 2000), "GB/s")
	fmt.Fprintln(w)

	fmt.Fprintf(w, "%s\n", ar.Note)
	fmt.Fprintf(w, "%s\n", idc.Note)
	fmt.Fprintf(w, "AllReduce 通信阶段 %.0f ms / 每 %.0f ms · IDC 每 ms 到达 %.0f 条小流\n",
		ar.CommMS, float64(p.StepMS), idc.Lambda)
}

// head 截断序列，火花线只画开头一段，否则每列跨度超过训练步周期就什么都看不出来。
func head(vals []float64, n int) []float64 {
	if len(vals) <= n {
		return vals
	}
	return vals[:n]
}

func winLabel(ms int) string {
	if ms >= 1000 {
		return fmt.Sprintf("%d s", ms/1000)
	}
	return fmt.Sprintf("%d ms", ms)
}

// WriteCSV 把两条曲线落盘，方便读者自己画图或改判读。
func WriteCSV(path string, idc, ar Traffic) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	b := &strings.Builder{}
	fmt.Fprintln(b, "t_ms,idc_gbps,allreduce_gbps")
	for i := range idc.GBps {
		v2 := 0.0
		if i < len(ar.GBps) {
			v2 = ar.GBps[i]
		}
		fmt.Fprintf(b, "%d,%.2f,%.2f\n", i, idc.GBps[i], v2)
	}
	_, err = io.WriteString(f, b.String())
	return err
}

var sparks = []rune("▁▂▃▄▅▆▇█")

func printSpark(w io.Writer, label string, vals []float64, unit string) {
	if len(vals) == 0 {
		return
	}
	const width = 96
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
	peak := maxOf(vals)
	if peak <= 0 {
		fmt.Fprintf(w, "%-20s 全程为 0\n", label)
		return
	}
	b := &strings.Builder{}
	for _, v := range cols {
		idx := int(v / peak * float64(len(sparks)-1))
		if v > 0 && idx == 0 {
			idx = 1
		}
		b.WriteRune(sparks[idx])
	}
	fmt.Fprintf(w, "%-20s 峰 %5.1f %s 均 %5.1f %s │%s│\n", label, peak, unit, meanOf(vals), unit, b.String())
}

func repeatChar(c rune, n int) string {
	out := make([]rune, n)
	for i := range out {
		out[i] = c
	}
	return string(out)
}
