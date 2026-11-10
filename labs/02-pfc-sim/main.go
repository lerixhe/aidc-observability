package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	cfg := DefaultConfig()
	flag.StringVar(&cfg.Mode, "mode", cfg.Mode, "流量模型：incast（AllReduce 同步突发）/ idc（传统小流）/ hol（队头阻塞）")
	flag.BoolVar(&cfg.PFC, "pfc", cfg.PFC, "是否启用 PFC 反压")
	flag.BoolVar(&cfg.DCQCN, "dcqcn", cfg.DCQCN, "是否启用 ECN 反馈闭环（简化版 DCQCN）")
	flag.IntVar(&cfg.Ports, "ports", cfg.Ports, "上游端口数")
	flag.Float64Var(&cfg.PortRate, "rate", cfg.PortRate, "端口线速 KB/μs（25 = 200 Gbps）")
	flag.Float64Var(&cfg.IngressBuf, "ingress-buf", cfg.IngressBuf, "每个 ingress 缓冲 KB")
	flag.Float64Var(&cfg.EgressBuf, "egress-buf", cfg.EgressBuf, "出端口共享缓冲 KB")
	flag.Float64Var(&cfg.SrcBuf, "src-buf", cfg.SrcBuf, "源端网卡发送积压上限 KB")
	flag.Float64Var(&cfg.XOFFRatio, "xoff", cfg.XOFFRatio, "XOFF 高水线占 ingress 缓冲比例")
	flag.Float64Var(&cfg.XONRatio, "xon", cfg.XONRatio, "XON 低水线占 ingress 缓冲比例")
	flag.Float64Var(&cfg.KminRatio, "kmin", cfg.KminRatio, "ECN 标记下限占出端口缓冲比例")
	flag.Float64Var(&cfg.KmaxRatio, "kmax", cfg.KmaxRatio, "ECN 标记上限占出端口缓冲比例")
	flag.Float64Var(&cfg.PauseQuota, "quota", cfg.PauseQuota, "单个 pause 帧的时间配额 μs")
	flag.Float64Var(&cfg.PauseDelay, "pause-delay", cfg.PauseDelay, "pause 帧传到上游的延迟 μs")
	flag.Float64Var(&cfg.Duration, "duration", cfg.Duration, "仿真时长 μs")
	flag.Float64Var(&cfg.SlowFactor, "slow", cfg.SlowFactor, "hol 模式下慢目的地可用带宽比例")
	flag.Float64Var(&cfg.Load, "load", cfg.Load, "idc 模式的平均负载，占出端口线速比例")
	flag.Float64Var(&cfg.BurstRatio, "burst", cfg.BurstRatio, "incast 模式下同步突发占周期的比例")
	flag.Float64Var(&cfg.Period, "period", cfg.Period, "incast 模式下的同步周期 μs")
	flag.BoolVar(&cfg.IgnorePause, "ignore-pause", false, "模拟上游忽略 pause 帧（PFC 配置不匹配），用来看 watchdog 介入")
	flag.Float64Var(&cfg.WatchdogUS, "watchdog", cfg.WatchdogUS, "PFC watchdog 阈值 μs，0 表示不启用")
	flag.Int64Var(&cfg.Seed, "seed", cfg.Seed, "随机种子（idc 模式用）")
	compare := flag.Bool("compare", false, "跑一组预设场景，输出横向对比表")
	outDir := flag.String("out", "out", "CSV 输出目录")
	flag.Parse()

	if err := validate(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "参数错误：%v\n", err)
		os.Exit(2)
	}

	if *compare {
		runCompare(cfg, *outDir)
		return
	}

	sim := NewSim(cfg)
	samples := sim.Run()
	Report(os.Stdout, cfg, sim.stats, samples)

	path := filepath.Join(*outDir, "timeseries.csv")
	if err := WriteCSV(path, samples); err != nil {
		fmt.Fprintf(os.Stderr, "写 CSV 失败：%v\n", err)
		os.Exit(1)
	}
	fmt.Printf("\n时序已写入 %s（%d 个采样点）\n", path, len(samples))
}

func validate(cfg Config) error {
	switch cfg.Mode {
	case "incast", "idc", "hol":
	default:
		return fmt.Errorf("未知 mode %q，可选 incast / idc / hol", cfg.Mode)
	}
	if cfg.Ports < 1 || cfg.Ports > 64 {
		return fmt.Errorf("ports 需在 1-64 之间")
	}
	if cfg.PortRate <= 0 || cfg.IngressBuf <= 0 || cfg.EgressBuf <= 0 {
		return fmt.Errorf("rate / ingress-buf / egress-buf 必须为正")
	}
	if !(cfg.XONRatio > 0 && cfg.XONRatio < cfg.XOFFRatio && cfg.XOFFRatio <= 1) {
		return fmt.Errorf("需满足 0 < xon < xoff <= 1")
	}
	if !(cfg.KminRatio >= 0 && cfg.KminRatio < cfg.KmaxRatio && cfg.KmaxRatio <= 1) {
		return fmt.Errorf("需满足 0 <= kmin < kmax <= 1")
	}
	if cfg.Duration <= 0 {
		return fmt.Errorf("duration 必须为正")
	}
	return nil
}

type scenario struct {
	name string
	cfg  Config
}

// runCompare 用同一套拓扑参数跑几种配置，把结论压成一张表。
func runCompare(base Config, outDir string) {
	scenarios := []scenario{
		{"idc 小流 · 无 PFC", with(base, func(c *Config) { c.Mode = "idc"; c.PFC = false })},
		{"incast · 无 PFC", with(base, func(c *Config) { c.Mode = "incast"; c.PFC = false })},
		{"incast · PFC", with(base, func(c *Config) { c.Mode = "incast"; c.PFC = true })},
		{"incast · PFC + DCQCN", with(base, func(c *Config) { c.Mode = "incast"; c.PFC = true; c.DCQCN = true })},
		{"incast · PFC，pause 延迟 20μs", with(base, func(c *Config) {
			c.Mode = "incast"
			c.PFC = true
			c.PauseDelay = 20
		})},
		{"hol · PFC", with(base, func(c *Config) { c.Mode = "hol"; c.PFC = true })},
	}

	fmt.Printf("== 02-pfc-sim · 场景对比 ==\n")
	fmt.Printf("拓扑 %.0f KB/μs × %d 上游 → 1 出端口 · ingress 缓冲 %.0f KB/口 · 出端口缓冲 %.0f KB · 时长 %.0f μs\n\n",
		base.PortRate, base.Ports, base.IngressBuf, base.EgressBuf, base.Duration)

	fmt.Printf("%-28s %9s %9s %8s %7s %8s %8s\n",
		"场景", "丢弃KB", "ECN%", "pause帧", "覆盖率", "利用率", "F2达成率")
	fmt.Println(repeatChar('-', 88))

	for _, sc := range scenarios {
		sim := NewSim(sc.cfg)
		sim.Run()
		st := sim.stats
		f2 := "-"
		if sc.cfg.Mode == "hol" && st.F2WantedKB > 0 {
			f2 = fmt.Sprintf("%.0f%%", safeDiv(st.F2SentKB, st.F2WantedKB)*100)
		}
		fmt.Printf("%-28s %9.0f %8.1f%% %8d %6.0f%% %7.0f%% %8s\n",
			sc.name,
			st.DroppedKB,
			safeDiv(st.ECNMarkedKB, st.SentKB)*100,
			st.PauseFrames,
			st.PauseCoverage(sc.cfg.Ports, sc.cfg.Duration)*100,
			safeDiv(st.SentKB, sc.cfg.PortRate*sc.cfg.Duration)*100,
			f2,
		)

		if err := WriteCSV(filepath.Join(outDir, "compare-"+slug(sc.name)+".csv"), sim.samples); err != nil {
			fmt.Fprintf(os.Stderr, "写 CSV 失败：%v\n", err)
		}
	}

	fmt.Println()
	fmt.Println("读法：")
	fmt.Println("  1. idc 那行零触发零丢弃 —— 老面板搬到智算网络看到的就是这个")
	fmt.Println("  2. incast 无 PFC 丢包、有 PFC 不丢但暂停成片 —— PFC 把丢包换成了堆积")
	fmt.Println("  3. PFC + DCQCN 的 pause 帧数应显著下降 —— ECN 是主防线，PFC 是最后防线")
	fmt.Println("  4. pause 延迟 20μs 那行开始丢包 —— pause 帧传播得越慢，缓冲越挡不住")
	fmt.Println("  5. hol 行看 F2 达成率 —— 无辜流被同队列的慢流一起冻住")
}

func with(c Config, f func(*Config)) Config {
	f(&c)
	return c
}

func repeatChar(c rune, n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = c
	}
	return string(b)
}

func slug(s string) string {
	// 只留 ASCII 字母数字，中文场景名统一退化为序号
	out := make([]rune, 0, len(s))
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			out = append(out, r)
		case r == ' ' || r == '-':
			out = append(out, '_')
		}
	}
	if len(out) == 0 {
		return "scenario"
	}
	return string(out)
}
