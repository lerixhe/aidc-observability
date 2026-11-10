// 11-case-study · 端到端排障剧本
//
// 这个 lab 不模拟网络，它模拟一次排障。十步，四把刀，四十分钟。
//
// 数据全部取自课程里那个「所有指标都正常」的案例：512 卡任务、
// rail-optimized 拓扑、光模块某条 lane 劣化。没有一条指标超阈值，
// 能发现问题的是斜率。
//
// 默认模式不需要 Docker：走一遍最优路径与一条蛮干路径，对比 MTTR。
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	run := flag.String("run", "both", "走哪条路径：best / naive / both")
	steps := flag.Bool("steps", false, "打印完整剧本（十步，含判据）")
	phase := flag.String("phase", "", "只看某一段：发现 / 定界 / 缩小 / 确认 / 恢复")
	live := flag.Bool("live", false, "打印真环境复刻方法（需要 03 与 05 两个 lab 的拓扑）")
	flag.Parse()

	if *live {
		ReportLive(os.Stdout)
		return
	}

	if *steps {
		ReportScript(os.Stdout, Script())
		return
	}

	best := Execute(BestPath())
	naive := Execute(NaivePath())

	switch strings.ToLower(*run) {
	case "best":
		reportOne(best, *phase)
	case "naive":
		reportOne(naive, *phase)
	case "both", "":
		ReportRun(os.Stdout, best)
		ReportMTTR(os.Stdout, best)
		ReportRun(os.Stdout, naive)
		ReportMTTR(os.Stdout, naive)
		ReportCompare(os.Stdout, best, naive)
	default:
		fmt.Fprintln(os.Stderr, "-run 只接受 best / naive / both")
		os.Exit(2)
	}
}

// reportOne 输出单条路径，可选只看某一段。
func reportOne(r Run, phase string) {
	steps := r.Path.Steps
	if phase != "" {
		steps = filterPhase(steps, Phase(phase))
		if len(steps) == 0 {
			fmt.Fprintf(os.Stderr, "路径 %s 里没有「%s」段的步骤\n", r.Path.Name, phase)
			os.Exit(2)
		}
	}
	sub := Execute(Path{Name: r.Path.Name, Steps: steps})
	ReportRun(os.Stdout, sub)
	ReportMTTR(os.Stdout, r)
}

func filterPhase(steps []Step, p Phase) []Step {
	var out []Step
	for _, s := range steps {
		if s.Phase == p {
			out = append(out, s)
		}
	}
	return out
}
