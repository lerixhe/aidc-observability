package main

import (
	"fmt"
	"io"
	"runtime"
	"strings"
)

// Report 打印体检报告。
func Report(w io.Writer, d *Discovery) {
	fmt.Fprintln(w, "== 00-setup · 环境自检 ==")
	fmt.Fprintf(w, "仓库 %s\n", d.RootDir)
	fmt.Fprintf(w, "平台 %s/%s · %d 核\n\n", runtime.GOOS, runtime.GOARCH, runtime.NumCPU())

	fmt.Fprintf(w, "%-16s %-12s %-6s %s\n", "工具", "版本", "状态", "说明")
	fmt.Fprintln(w, repeatChar('-', 64))
	printTool(w, d.Go, 1, 22)
	printTool(w, d.Docker, 0, 0)
	printTool(w, d.Clab, 0, 0)
	printTool(w, d.Python, 0, 0)
	fmt.Fprintln(w)

	ready, total := 0, len(d.Labs)
	for _, lab := range d.Labs {
		if lab.Ready {
			ready++
		}
	}
	fmt.Fprintf(w, "lab 盘点（%d 个已落地 / 共登记 %d 个）\n", ready, total)
	fmt.Fprintf(w, "  %-18s %-4s %-20s %-8s %-6s %s\n", "目录", "层", "需要工具", "对应课", "可跑", "说明")
	fmt.Println(repeatChar('-', 96))
	for _, lab := range d.Labs {
		canRun, missing := d.CanRun(lab)
		mark, note := "✅", lab.Summary
		switch {
		case !lab.Ready:
			mark, note = "—", "规划中 · "+lab.Summary
		case !canRun:
			mark = "❌"
			note = fmt.Sprintf("缺 %s · %s", strings.Join(missing, "+"), lab.Summary)
		}
		fmt.Fprintf(w, "  %-18s %-4s %-20s %-8s %-6s %s\n",
			lab.Dir, lab.Layer, shortNeeds(lab.Needs), lab.Lesson, mark, note)
	}
	if len(d.Unlisted) > 0 {
		fmt.Fprintf(w, "\n未登记的目录：%s（加到 labs/00-setup/probe.go 的 knownLabs）\n",
			strings.Join(d.Unlisted, ", "))
	}

	fmt.Fprintln(w)
	printNextSteps(w, d)
}

func printTool(w io.Writer, t ToolStatus, wantMajor, wantMinor int) {
	mark := "✅"
	detail := t.Detail
	if t.OK {
		if wantMajor > 0 {
			v := parseSemver(t.Version)
			if v[0] < wantMajor || (v[0] == wantMajor && v[1] < wantMinor) {
				mark = "❌"
				detail = fmt.Sprintf("版本过低，需要 >= %d.%d", wantMajor, wantMinor)
			}
		}
	} else {
		mark = "❌"
	}
	fmt.Fprintf(w, "%-16s %-12s %-6s %s\n", t.Name, orDash(t.Version), mark, detail)
}

// shortNeeds 把工具名压短，避免表格被 05-clos-topo 那种长依赖撑破。
func shortNeeds(needs []string) string {
	out := make([]string, len(needs))
	for i, n := range needs {
		if n == "containerlab" {
			n = "clab"
		}
		if n == "python3" {
			n = "py3"
		}
		out[i] = n
	}
	return strings.Join(out, "+")
}

func orDash(s string) string {
	if s == "" || s == "未知" {
		return "—"
	}
	return s
}

func printNextSteps(w io.Writer, d *Discovery) {
	fmt.Fprintln(w, "下一步")
	if !d.GoAtLeast(1, 22) {
		fmt.Fprintln(w, "  先装 Go 1.22 以上：https://go.dev/dl/")
		return
	}
	var runnable []Lab
	for _, lab := range d.Labs {
		if lab.Ready {
			if ok, _ := d.CanRun(lab); ok {
				runnable = append(runnable, lab)
			}
		}
	}
	if len(runnable) == 0 {
		fmt.Fprintln(w, "  还没有可跑的 L1 lab")
		return
	}
	first := runnable[0]
	if len(runnable) > 1 {
		// 02-pfc-sim 是核心样本，优先推荐
		for _, lab := range runnable {
			if lab.Dir == "02-pfc-sim" {
				first = lab
			}
		}
	}
	fmt.Fprintf(w, "  1. cd labs/%s && make run\n", first.Dir)
	fmt.Fprintln(w, "  2. 想一次跑完所有检查：go run . -try")
	if !d.Docker.OK {
		fmt.Fprintln(w, "  3. Docker 缺失不影响 L1 lab，只影响 v0.4 的 L2 拓扑实验")
	}
}

func repeatChar(c rune, n int) string {
	out := make([]rune, n)
	for i := range out {
		out[i] = c
	}
	return string(out)
}
