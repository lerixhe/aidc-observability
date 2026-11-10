package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	p := DefaultParams()
	flag.IntVar(&p.DurationMS, "duration-ms", p.DurationMS, "观测时长，ms")
	flag.Float64Var(&p.MeanGBps, "mean", p.MeanGBps, "目标平均负载，GB/s（两种模型对齐到同一个值）")
	flag.Float64Var(&p.PortGBps, "port", p.PortGBps, "单端口线速，GB/s（25 = 200 Gbps）")
	flag.IntVar(&p.Workers, "workers", p.Workers, "AllReduce 并发 worker 数")
	flag.Float64Var(&p.FlowMedianKB, "flow-median-kb", p.FlowMedianKB, "IDC 流大小中位数，KB")
	flag.Float64Var(&p.FlowSigma, "flow-sigma", p.FlowSigma, "IDC 流大小对数正态 sigma，控制长尾")
	flag.IntVar(&p.StepMS, "step-ms", p.StepMS, "训练步周期，ms")
	flag.Int64Var(&p.Seed, "seed", p.Seed, "随机种子")
	outDir := flag.String("out", "out", "CSV 输出目录")
	flag.Parse()

	if err := validate(p); err != nil {
		fmt.Fprintf(os.Stderr, "参数错误：%v\n", err)
		os.Exit(2)
	}

	idc := idcTraffic(p)
	ar := allreduceTraffic(p)
	Report(os.Stdout, p, idc, ar)

	path := filepath.Join(*outDir, "traffic.csv")
	if err := WriteCSV(path, idc, ar); err != nil {
		fmt.Fprintf(os.Stderr, "写 CSV 失败：%v\n", err)
		os.Exit(1)
	}
	fmt.Printf("\n时序已写入 %s（%d 个 1ms 采样点）\n", path, len(idc.GBps))
}

func validate(p Params) error {
	if p.DurationMS < 1000 {
		return fmt.Errorf("duration-ms 至少 1000，否则看不出训练步的节奏")
	}
	if p.MeanGBps <= 0 || p.PortGBps <= 0 {
		return fmt.Errorf("mean / port 必须为正")
	}
	if p.Workers < 1 {
		return fmt.Errorf("workers 至少为 1")
	}
	if p.StepMS < 10 || p.StepMS > p.DurationMS/2 {
		return fmt.Errorf("step-ms 需在 10 到 duration-ms/2 之间")
	}
	if p.FlowMedianKB <= 0 {
		return fmt.Errorf("flow-median-kb 必须为正")
	}
	comm := float64(p.StepMS) * p.MeanGBps / (float64(p.Workers) * p.PortGBps)
	if comm <= 0 || comm >= float64(p.StepMS) {
		return fmt.Errorf("参数组合导致通信阶段 %.1f ms 超出周期，调小 mean 或调大 workers/port", comm)
	}
	return nil
}
