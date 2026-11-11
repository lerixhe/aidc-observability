package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	trials := flag.Int("trials", 200, "随机故障的轮数")
	seed := flag.Int64("seed", 42, "随机种子")
	gap := flag.Duration("gap", 150*time.Millisecond, "告警收敛的时间窗口")
	window := flag.Duration("window", 10*time.Second, "告警流的时间跨度")
	topo := flag.Bool("topo", false, "只打印拓扑，不跑评估")
	format := flag.String("format", "text", "输出格式：text / dot")
	flag.Parse()

	if *topo {
		t := BuildClos(2, 4, 2)
		if *format == "dot" {
			ReportDot(os.Stdout, t, nil, Truth{})
			return
		}
		fmt.Println("节点与下游集合：")
		for _, n := range t.Nodes {
			fmt.Printf("  %-8s %-6s down=%v\n", n.Name, n.Tier, t.Down[n.Name])
		}
		return
	}

	if *trials < 1 {
		fmt.Fprintln(os.Stderr, "trials 至少为 1")
		os.Exit(2)
	}

	e := Run(*trials, *seed, *gap, *window)
	if *format == "dot" {
		t := BuildClos(2, 4, 2)
		if e.Sample != nil {
			ReportDot(os.Stdout, t, e.Sample.Graph, e.Sample.Truth)
		} else {
			ReportDot(os.Stdout, t, nil, Truth{})
		}
		return
	}
	Report(os.Stdout, e)

	fmt.Println()
	fmt.Println("读法：")
	fmt.Println("  1. 按告警条数排序最差 —— 一个故障的派生告警最多，根因往往不是告警最多的那个")
	fmt.Println("  2. 按最早时间排序中等 —— 会被上游反向告警和早到的噪声带偏")
	fmt.Println("  3. 图打分靠 precision 压住「覆盖面最大的顶层节点」，这才是共同祖先搜索的关键")
	fmt.Println("  4. 压缩 6 倍左右：四千条告警压成六百多个 incident，RCA 才有得算")
}
