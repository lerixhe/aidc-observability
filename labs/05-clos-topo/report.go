package main

import (
	"fmt"
	"io"
	"strings"
)

// 三个实验的输出。格式固定，数字格式固定，锁进 expected/ 供 CI 比对。

// ReportTopology 打印拓扑与地址表，读者照着敲路由。
func ReportTopology(w io.Writer) {
	fmt.Fprintln(w, "== 拓扑：2 spine / 4 leaf / 8 host ==")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "        spine1        spine2")
	fmt.Fprintln(w, "          │ │           │ │")
	fmt.Fprintln(w, "   ┌──────┴─┴───────────┴─┴──────┐")
	fmt.Fprintln(w, " leaf1  leaf2  leaf3  leaf4")
	fmt.Fprintln(w, "   │      │      │      │")
	fmt.Fprintln(w, " h1 h2  h3 h4  h5 h6  h7 h8")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "主机                leaf    地址")
	for _, h := range Hosts() {
		fmt.Fprintf(w, "%-4s %-15s %-6s  %s\n", h.Name, "", h.Leaf, ipString(h.IP))
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "上行链路（每条 leaf 两条等价路径，均分时各 50%）")
	fmt.Fprintln(w, "leaf   接口   spine    本端地址        对端地址")
	for _, u := range Uplinks() {
		fmt.Fprintf(w, "%-6s %-5s  %-7s  %-14s  %s\n",
			u.Leaf, u.Iface, u.Spine, ipString(u.Local), ipString(u.Remote))
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "两端口径会决定这八条上行吃多少流量，这就是本 lab 要量的东西。")
	fmt.Fprintln(w)
}

// ReportExperiment1 打印流数扫描：大数定律在散列上的表现。
func ReportExperiment1(w io.Writer, seed int64, counts []int) {
	fmt.Fprintf(w, "== 实验一：流数怎么影响散列均匀度（policy=L4，随机 5 元组，seed=%d）==\n", seed)
	fmt.Fprintln(w, "流数    leaf1     leaf2     leaf3     leaf4     最差极化度  CV     判定")
	for _, n := range counts {
		res := Simulate(Generate(WorkRandom, n, seed), PolicyL4)
		fmt.Fprintf(w, "%-7d %-9s %-9s %-9s %-9s %-11.2f %-6.2f %s\n",
			n, cell(res, "leaf1"), cell(res, "leaf2"), cell(res, "leaf3"), cell(res, "leaf4"),
			res.Worst, res.CV, verdict(res.Worst))
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "读法：")
	fmt.Fprintln(w, "  1. 流数是散列均匀度的第一变量。八条流时两条上行只用上一条，不奇怪")
	fmt.Fprintln(w, "  2. 极化度 1.0 是完美均分，2.0 是「两条上行只用到一条」的极端")
	fmt.Fprintln(w, "  3. 少流的瓶颈不是算法，是样本量：你在用八次抽样去逼近 50%")
	fmt.Fprintln(w)
}

// ReportExperiment2 打印口径对照：三层与五元组的差别。
func ReportExperiment2(w io.Writer, n int, seed int64) {
	fmt.Fprintf(w, "== 实验二：hash 口径（%d 条流，源目 IP 固定、端口递增）==\n", n)
	fmt.Fprintln(w, "口径                最差极化度  CV     最大链路占比  说明")
	for _, p := range []struct {
		pol  HashPolicy
		name string
		note string
	}{
		{PolicyL3, "L3（src/dst IP）", "端口不参与，同一对主机的连接全撞一条上行"},
		{PolicyL4, "L4（五元组）", "端口参与，连接被摊开，但流数仍决定摊得多平"},
	} {
		res := Simulate(Generate(WorkSameTuple, n, seed), p.pol)
		fmt.Fprintf(w, "%-19s %-11.2f %-6.2f %-13s %s\n",
			p.name, res.Worst, res.CV, percent(maxShare(res)), p.note)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "读法：")
	fmt.Fprintln(w, "  1. 三层口径下不是「散得不好」，是根本没用端口这一维，等价于只有一条路")
	fmt.Fprintln(w, "  2. 内核里由 net.ipv4.fib_multipath_hash_policy 控制：0 走三层，1 走四层")
	fmt.Fprintln(w, "  3. 默认值取决于发行版与内核版本，接手集群时先确认这个开关，别猜")
	fmt.Fprintln(w)
}

// ReportExperiment3 打印同质大象流：hash 救不了的场合。
func ReportExperiment3(w io.Writer, n int, seed int64) {
	fmt.Fprintf(w, "== 实验三：同质大象流（%d 条流，源目 IP 与端口全部相同）==\n", n)
	fmt.Fprintln(w, "口径                最差极化度  CV     最大链路占比")
	for _, p := range []struct {
		pol  HashPolicy
		name string
	}{
		{PolicyL3, "L3（src/dst IP）"},
		{PolicyL4, "L4（五元组）"},
	} {
		res := Simulate(Generate(WorkElephant, n, seed), p.pol)
		fmt.Fprintf(w, "%-19s %-11.2f %-6.2f %s\n", p.name, res.Worst, res.CV, percent(maxShare(res)))
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "读法：")
	fmt.Fprintln(w, "  1. 五元组一模一样时，两种口径都判同一条路。这不是算法问题，是输入没有新信息")
	fmt.Fprintln(w, "  2. 训练流量恰好长这样：rail 对齐的 AllReduce，5 元组在整个训练周期不变")
	fmt.Fprintln(w, "  3. 所以智算网络靠结构而不是靠 hash：rail-optimized 拓扑把流量预分到不同轨道")
	fmt.Fprintln(w, "  4. hash 能做的是「多流之间摊平」，做不了「一条流自己分叉」。flowlet 与自适应路由才做后者")
	fmt.Fprintln(w)
}

// cell 返回某 leaf 的两条上行分布，形如 5:3。
func cell(r Result, leaf string) string {
	for _, ls := range r.Leaves {
		if ls.Leaf == leaf {
			return ls.Ratio()
		}
	}
	return "-"
}

// maxShare 返回全局最大链路占比。
func maxShare(r Result) float64 {
	var m float64
	for _, ls := range r.Leaves {
		for _, l := range ls.Links {
			if l.Share > m {
				m = l.Share
			}
		}
	}
	return m
}

// verdict 给极化度一个中文判定，阈值写在 README 里。
func verdict(polar float64) string {
	switch {
	case polar >= 1.4:
		return "严重不均"
	case polar >= 1.15:
		return "明显不均"
	}
	return "可接受"
}

func percent(v float64) string {
	return strings.TrimSuffix(fmt.Sprintf("%.1f", v*100), ".0") + "%"
}

func ipString(ip uint32) string {
	return fmt.Sprintf("%d.%d.%d.%d", byte(ip>>24), byte(ip>>16), byte(ip>>8), byte(ip))
}
