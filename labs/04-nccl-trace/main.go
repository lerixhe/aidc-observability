package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func main() {
	logPath := flag.String("log", "samples/nccl-ring8.log", "要解析的 NCCL 日志")
	portGBps := flag.Float64("port", 25, "单端口线速 GB/s，用于算效率与预期耗时")
	genPath := flag.String("gen", "", "生成一份教学日志到指定文件后退出（默认样本见 samples/）")

	gp := DefaultGenParams()
	flag.IntVar(&gp.Ranks, "ranks", gp.Ranks, "生成日志的 rank 数")
	flag.IntVar(&gp.Steps, "steps", gp.Steps, "生成日志的训练步数")
	flag.Float64Var(&gp.PortGBps, "gen-port", gp.PortGBps, "生成日志时的单端口线速 GB/s")
	flag.Float64Var(&gp.Jitter, "jitter", gp.Jitter, "正常抖动的标准差")
	flag.Int64Var(&gp.Seed, "seed", gp.Seed, "随机种子")

	msgMB := flag.Float64("msg-mb", float64(gp.MsgBytes)/(1<<20), "每步 allReduce 消息大小 MB（仅 -gen 用）")
	fault := flag.String("fault", "", "注入 straggler，格式 step_from-step_to,rank,scale，如 7-9,3,2.5")
	ringStr := flag.String("ring", "", "自定义环顺序，如 0,2,4,6,1,3,5,7")
	flag.Parse()

	gp.MsgBytes = *msgMB * (1 << 20)
	if *ringStr != "" {
		r, err := parseRingList(*ringStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ring 参数错误：%v\n", err)
			os.Exit(2)
		}
		gp.Ring = r
	}
	noFault := flag.Bool("no-fault", false, "生成日志时不注入 straggler")
	if *fault != "" {
		fs, rank, scale, err := parseFault(*fault)
		if err != nil {
			fmt.Fprintf(os.Stderr, "fault 参数错误：%v\n", err)
			os.Exit(2)
		}
		gp.FaultStep, gp.FaultRank, gp.FaultScale = fs, rank, scale
	}
	if *noFault {
		gp.FaultRank = -1
	}

	if *genPath != "" {
		text := GenerateLog(gp)
		if err := os.WriteFile(*genPath, []byte(text), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "写日志失败：%v\n", err)
			os.Exit(1)
		}
		fmt.Printf("已生成 %s（%d rank × %d 步）\n", *genPath, gp.Ranks, gp.Steps)
		return
	}

	f, err := os.Open(*logPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "打开日志失败：%v\n（可先跑 make gen 生成样本）\n", err)
		os.Exit(1)
	}
	defer f.Close()

	l, err := Parse(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "解析失败：%v\n", err)
		os.Exit(1)
	}
	Report(os.Stdout, l, Analyze(l), *logPath, *portGBps)
}

func parseRingList(s string) ([]int, error) {
	parts := strings.Split(s, ",")
	ring := make([]int, 0, len(parts))
	for _, p := range parts {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			return nil, fmt.Errorf("无法解析 %q", p)
		}
		ring = append(ring, n)
	}
	if len(ring) < 2 {
		return nil, fmt.Errorf("环至少要有 2 个 rank")
	}
	return ring, nil
}

func parseFault(s string) ([2]int, int, float64, error) {
	parts := strings.Split(s, ",")
	if len(parts) != 3 {
		return [2]int{}, 0, 0, fmt.Errorf("需要三段：step_from-step_to,rank,scale")
	}
	rng := strings.Split(parts[0], "-")
	if len(rng) != 2 {
		return [2]int{}, 0, 0, fmt.Errorf("步区间要写成 from-to")
	}
	from, err1 := strconv.Atoi(strings.TrimSpace(rng[0]))
	to, err2 := strconv.Atoi(strings.TrimSpace(rng[1]))
	rank, err3 := strconv.Atoi(strings.TrimSpace(parts[1]))
	scale, err4 := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
	if err := firstErr(err1, err2, err3, err4); err != nil {
		return [2]int{}, 0, 0, err
	}
	if to < from || scale < 1 {
		return [2]int{}, 0, 0, fmt.Errorf("要求 to >= from 且 scale >= 1")
	}
	return [2]int{from, to}, rank, scale, nil
}

func firstErr(errs ...error) error {
	for _, e := range errs {
		if e != nil {
			return e
		}
	}
	return nil
}
