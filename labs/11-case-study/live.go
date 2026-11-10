package main

import (
	"fmt"
	"io"
)

// 剧本能不能接到真环境？逐条回答。
//
// 十步里有六步能用容器拓扑复刻，另外四步要真硬件：光模块的 lane、
// PFC 的 per-priority 计数、INT 的逐跳元数据、DOM 读数。
// 把能复刻的复刻，不能复刻的指出来 —— 这比假装全都能跑更有用。

// LiveStep 描述一步在真实环境里怎么落。
type LiveStep struct {
	ID     int
	How    string
	Where  string // lab 目录 + 命令
	CanRun bool   // 容器里能不能复刻
}

// LivePlan 返回十步的真环境落法。
func LivePlan() []LiveStep {
	return []LiveStep{
		{1, "读训练日志的步耗时分布", "训练侧，无需 lab", false},
		{2, "算 allreduce 耗时占比", "训练侧，无需 lab", false},
		{3, "查降频 / XID / NVLink 重放", "主机侧，无需 lab", false},
		{4, "看 NCCL ring 组成是否变化", "labs/04-nccl-trace 可解析日志", true},
		{5, "按优先级看 pause 时长，逐跳收窄", "labs/03-fault-inject 的 observe 流程", true},
		{6, "看 ECN 标记率斜率与队列峰值", "labs/05-clos-topo 的 collect.sh 采上行计数", true},
		{7, "INT 逐跳时延分解", "labs/07-int-path 能编解码元数据，但需真交换机落地", true},
		{8, "CRC 与 pre-FEC BER 的一阶差分", "labs/03-fault-inject 里用 netem corrupt 造错误率", true},
		{9, "逐 lane 读 DOM 光功率", "只能真机：容器里没有 DOM（数字诊断监控）", false},
		{10, "drain 端口、验证、换件、回切", "labs/05-clos-topo 里用路由撤换模拟 drain", true},
	}
}

// ReportLive 打印真环境复刻方法。
func ReportLive(w io.Writer) {
	fmt.Fprintln(w, "== 真环境复刻：十步里能跑几步 ==")
	fmt.Fprintln(w, "步  能不能复刻  做什么                           落在哪")
	fmt.Fprintln(w, "--  ----------  -------------------------------  ------------------------------")
	ok := 0
	for _, l := range LivePlan() {
		mark := "不能"
		if l.CanRun {
			mark = "能"
			ok++
		}
		fmt.Fprintf(w, "%-2d  %-10s  %s  %s\n", l.ID, pad(mark, 10), pad(l.How, 31), l.Where)
	}
	fmt.Fprintf(w, "\n%d / %d 步可以在容器拓扑里复刻。剩下四步缺的都是硬件读数，不是方法。\n\n", ok, len(LivePlan()))

	fmt.Fprintln(w, "具体怎么接：")
	fmt.Fprintln(w, "  1. 起拓扑：cd labs/05-clos-topo && make deploy")
	fmt.Fprintln(w, "  2. 造故障：cd labs/03-fault-inject && make inject S=hol")
	fmt.Fprintln(w, "  3. 采数据：cd labs/05-clos-topo && make collect")
	fmt.Fprintln(w, "  4. 判读：  cd labs/03-fault-inject && go run . -classify -dir <你的样本目录>")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "照这个顺序走一遍，第 5、6、8 步的数据就有了。第 7 步需要交换机支持 INT，")
	fmt.Fprintln(w, "第 9 步必须有真实光模块 —— 这两步只能在生产环境里练。")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "一句提醒：容器拓扑里采到的是真计数器（tx_bytes、队列 backlog、丢弃），")
	fmt.Fprintln(w, "但 pause 与 DOM 是模型补的。混着看的时候，先确认每一列从哪来。")
}
