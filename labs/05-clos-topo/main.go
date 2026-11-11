// 05-clos-topo · ECMP 散列极化
//
// 一个反直觉的事实：Clos 上联做到 1:1 无收拢，仍然可能有一条上行打满、
// 另一条空着。原因不在带宽，在散列：数据面按 5 元组取 hash 再对链路数取模，
// 流少的时候，取模的结果就是不公平。
//
// 默认模式不需要 Docker：算散列分布、给结论。真拓扑在 topo.clab.yml，
// 用 make deploy 起来后可以采真实计数器对照（README 第 5 节）。
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func main() {
	topo := flag.Bool("topo", false, "只打印拓扑与地址表")
	format := flag.String("format", "text", "输出格式：text / mermaid")
	flows := flag.Int("flows", 64, "单次模拟的流数")
	kind := flag.String("kind", "random", "流形态：random / elephant / sametuple")
	policy := flag.String("policy", "l4", "hash 口径：l3 / l4")
	seed := flag.Int64("seed", 42, "随机种子")
	exp := flag.Int("exp", 0, "只跑某个实验（1/2/3），0 表示全跑")
	scan := flag.String("scan", "8,32,128,512,2048", "实验一的流数序列")
	flag.Parse()

	if *topo {
		if *format == "mermaid" {
			ReportMermaid(os.Stdout, Result{})
			return
		}
		ReportTopology(os.Stdout)
		fmt.Fprintln(os.Stdout, "真拓扑：make deploy   # containerlab deploy -t topo.clab.yml")
		fmt.Fprintln(os.Stdout, "清理：  make destroy")
		return
	}

	counts, err := parseScan(*scan)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	switch *exp {
	case 0:
		ReportExperiment1(os.Stdout, *seed, counts)
		ReportExperiment2(os.Stdout, *flows, *seed)
		ReportExperiment3(os.Stdout, *flows, *seed)
	case 1:
		ReportExperiment1(os.Stdout, *seed, counts)
	case 2:
		ReportExperiment2(os.Stdout, *flows, *seed)
	case 3:
		ReportExperiment3(os.Stdout, *flows, *seed)
	default:
		fmt.Fprintln(os.Stderr, "-exp 只接受 1、2、3，或 0 表示全跑")
		os.Exit(2)
	}

	// -kind 与 -policy 只影响单次明细输出，实验表格自带口径
	if *exp == 0 {
		return
	}

	p, err := parsePolicy(*policy)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	work := Workload(*kind)
	res := Simulate(Generate(work, *flows, *seed), p)

	if *format == "mermaid" {
		ReportMermaid(os.Stdout, res)
		return
	}

	ReportDetail(os.Stdout, res, *kind)
}

// ReportDetail 打印一次模拟的逐链路明细。放在 main 里，因为它只服务单次查询。
func ReportDetail(w *os.File, res Result, kind string) {
	fmt.Fprintf(w, "== 明细：%d 条流 · 口径 %s · 形态 %s ==\n",
		res.Total, policyName(res.Policy), kind)
	fmt.Fprintf(w, "其中 %d 条同 leaf 内互访（不走上行），%d 条经上行到 spine\n\n",
		res.Local, res.Total-res.Local)
	fmt.Fprintln(w, "链路              流数   占比")
	for _, ls := range res.Leaves {
		for _, l := range ls.Links {
			fmt.Fprintf(w, "%-17s %-5d  %s\n", l.Link, l.Flows, percent(l.Share))
		}
	}
	fmt.Fprintf(w, "\n最差极化度 %.2f · 链路占比变异系数 %.2f\n", res.Worst, res.CV)
}

func policyName(p HashPolicy) string {
	if p == PolicyL4 {
		return "L4（五元组）"
	}
	return "L3（三层）"
}

func parsePolicy(s string) (HashPolicy, error) {
	switch strings.ToLower(s) {
	case "l3", "3", "0":
		return PolicyL3, nil
	case "l4", "4", "1":
		return PolicyL4, nil
	}
	return PolicyL3, fmt.Errorf("不认识的口径 %q，可选 l3 / l4", s)
}

func parseScan(s string) ([]int, error) {
	var out []int
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		n, err := strconv.Atoi(part)
		if err != nil || n <= 0 {
			return nil, fmt.Errorf("流数序列 %q 里有非法值 %q", s, part)
		}
		out = append(out, n)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("流数序列为空")
	}
	return out, nil
}
