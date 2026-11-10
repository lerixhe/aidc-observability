package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	scenario := flag.String("scenario", "all", "场景：healthy / egress / upstream / forwarding / all")
	k := flag.Int("sample", 2048, "采样率 1:k，只影响 CSV 里的说明，不影响判读")
	pktRate := flag.Int("pkt-rate", DefaultPktRate, "报文速率（报文/秒），用于采样对照")
	burstUs := flag.Int("burst-us", DefaultBurstUs, "微突发持续时长（微秒）")
	seed := flag.Int64("seed", 42, "采样对照的随机种子")
	trials := flag.Int("trials", 2000, "采样对照的蒙特卡洛轮数")
	csv := flag.Bool("csv", false, "把逐跳明细写到 out/hops.csv")
	flag.Parse()

	var scenarios []Scenario
	if *scenario == "all" {
		scenarios = Scenarios
	} else {
		sc, err := ParseScenario(*scenario)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		scenarios = []Scenario{sc}
	}

	paths := make([]Path, 0, len(scenarios))
	an := make([]Analysis, 0, len(scenarios))
	for _, sc := range scenarios {
		p := BuildPath(sc)
		paths = append(paths, p)
		// 走一遍线格式再解析，确保判读用的是解码结果而不是内存里的原值
		hops, err := Decode(p.Payload())
		if err != nil {
			fmt.Fprintf(os.Stderr, "解码失败：%v\n", err)
			os.Exit(1)
		}
		an = append(an, Analyze(p.FlowID, hops))
	}

	rows := RunSampling(*pktRate, *burstUs, *seed, *trials)
	Report(os.Stdout, paths, an, rows, *pktRate, *burstUs)

	if *csv {
		dir := "out"
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "建目录失败：%v\n", err)
			os.Exit(1)
		}
		path := filepath.Join(dir, "hops.csv")
		f, err := os.Create(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "写 CSV 失败：%v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		for _, a := range an {
			fmt.Fprint(f, CSV(a))
		}
		fmt.Printf("\n逐跳明细已写入 %s（采样率 1:%d 只影响真实上报量）\n", path, *k)
	}

	fmt.Println()
	fmt.Println("读法：")
	fmt.Println("  1. healthy 场景全跳 ok —— 判读逻辑不会把正常路径认成故障")
	fmt.Println("  2. egress 场景利用率 97% 加队列 780 KB —— 结论是这一跳的出口在排队")
	fmt.Println("  3. upstream 场景队列堆着但出口只有 55% —— 问题在上游分布，不在这一跳")
	fmt.Println("  4. forwarding 场景队列空、出口闲而停留 40 µs —— 开销在转发面，调水线没用")
	fmt.Println("  5. 采样表：1:4096 时一次 200 µs 微突发的捕获率只有个位数，别拿它当监控")
}
