package main

// 一份 40 分钟的排障剧本，十个步骤，四把刀。
//
// 数据来自课程里那个「所有指标都正常」的案例：512 卡任务、rail-optimized 拓扑、
// 光模块某一条 lane 劣化。所有单点指标都在阈值内，能发现问题的是斜率。
//
// 每一步都定死四件事：看什么、看到什么、判据是什么、这一步花多久。
// 花多久这件事最容易被忽略，也最影响 MTTR —— 第 19 课把它拆成五段。

// Phase 是 MTTR 的五段之一。顺序固定，累加时按这个顺序输出。
type Phase string

const (
	PhaseDetect  Phase = "发现"
	PhaseBound   Phase = "定界"
	PhaseNarrow  Phase = "缩小"
	PhaseConfirm Phase = "确认"
	PhaseRecover Phase = "恢复"
)

// PhaseOrder 决定汇报与累加的顺序。
var PhaseOrder = []Phase{PhaseDetect, PhaseBound, PhaseNarrow, PhaseConfirm, PhaseRecover}

// Step 是剧本里的一步。
type Step struct {
	ID        int
	Phase     Phase
	Question  string  // 这一步要回答的问题
	Look      string  // 看什么数据
	Seen      string  // 观测到什么（模拟读数）
	Criterion string  // 判据
	Judgment  string  // 得到什么结论
	Cost      float64 // 耗时，分钟
}

// Script 返回十条最优路径的步骤，顺序即执行顺序。
func Script() []Step {
	return []Step{
		{
			ID: 1, Phase: PhaseDetect,
			Question:  "全体还是个别的慢",
			Look:      "训练日志的步耗时分布",
			Seen:      "步耗时 380 → 520 ms；512 个 rank 同步抬升，无掉队者；P50 与 P99 同向",
			Criterion: "全体同步 → 共享资源（网络 / 存储 / 调度）优先；个别慢 → 先查那台主机",
			Judgment:  "全体同步，网络层优先",
			Cost:      1.0,
		},
		{
			ID: 2, Phase: PhaseBound,
			Question:  "时间被谁吃掉了",
			Look:      "日志里 allreduce 耗时与占比",
			Seen:      "allreduce 90 → 190 ms；通信占比 24% → 36%",
			Criterion: "占比上升 → 通信问题；占比不变而总时长涨 → 算力侧",
			Judgment:  "通信吃掉了增量，GPU 侧先排除",
			Cost:      1.5,
		},
		{
			ID: 3, Phase: PhaseBound,
			Question:  "GPU 与域内网络是否干净",
			Look:      "nvidia-smi 降频原因、dmesg 的 XID、DCGM 的 NVLink 重放",
			Seen:      "三项全干净：无降频、无 XID、NVLink 无重放",
			Criterion: "干净 → 排除算力与域内；有 XID 或重放 → 先处理主机侧",
			Judgment:  "排除单机与 NVLink",
			Cost:      2.0,
		},
		{
			ID: 4, Phase: PhaseBound,
			Question:  "通信环还在原路线上吗",
			Look:      "NCCL_DEBUG=INFO SUBSYS=INIT,GRAPH 的 ring 组成",
			Seen:      "ring 与基线一致，未跨 rail 重组",
			Criterion: "ring 变了 → 环重组会带来全体变慢，而网络计数器正常",
			Judgment:  "环没变，网络指标对得上号",
			Cost:      1.5,
		},
		{
			ID: 5, Phase: PhaseNarrow,
			Question:  "哪台设备的哪个端口在冒烟",
			Look:      "gNMI 订阅的 per-priority pause 时长，10 分钟窗口",
			Seen:      "只有 rail-3 的 leaf-7 上行口异常：累计 pause 42 秒，占 7%",
			Criterion: "> 10 秒 / 10 分钟为严重；只看时长不看帧数",
			Judgment:  "候选从全网压到单个端口",
			Cost:      4.0,
		},
		{
			ID: 6, Phase: PhaseNarrow,
			Question:  "拥塞是被压制还是在恶化",
			Look:      "同一端口的 ECN 标记率斜率与队列峰值",
			Seen:      "ECN 0.4% → 3.8%，十倍级上升；队列峰值未顶满",
			Criterion: "ECN 持续 > 3% 即异常；队列未顶满说明还没到风暴",
			Judgment:  "机制在工作，但压力在持续上升",
			Cost:      2.0,
		},
		{
			ID: 7, Phase: PhaseNarrow,
			Question:  "逐跳看，卡在哪一跳",
			Look:      "INT 逐跳时延与队列元数据",
			Seen:      "时延集中在 leaf-7 上行口的出口队列，其余各跳正常",
			Criterion: "集中在某一跳的出口 → 该跳下游能力不足，而不是上游打得太猛",
			Judgment:  "问题在该端口往外的那一段",
			Cost:      6.0,
		},
		{
			ID: 8, Phase: PhaseConfirm,
			Question:  "链路底层有没有出错",
			Look:      "端口 CRC 计数与 pre-FEC BER 的一阶差分",
			Seen:      "CRC 0 → 12 次/分钟；FEC 纠正率 1e-9 → 4e-6",
			Criterion: "绝对值不重要，趋势重要：从 0 起涨就是第一阶段证据",
			Judgment:  "链路在硬扛，纠错压力上升",
			Cost:      4.0,
		},
		{
			ID: 9, Phase: PhaseConfirm,
			Question:  "哪个 lane 出了问题",
			Look:      "模块 DOM 的逐 lane 接收光功率",
			Seen:      "lane 2 为 -8.9 dBm，其余 lane 在 -2.4 dBm 附近",
			Criterion: "单 lane 偏离 > 3 dB 即劣化；偏离 6 dB 不是抖动",
			Judgment:  "lane 级劣化，四个证据指向同一处",
			Cost:      5.0,
		},
		{
			ID: 10, Phase: PhaseRecover,
			Question:  "止血、验证、根治、回切",
			Look:      "drain 端口后的步耗时、ring 重组、回切结果",
			Seen:      "drain 后步耗时回到 390 ms；流量绕开；换模块后回切并登记根因",
			Criterion: "步耗时回到基线 ±10% 才算恢复；不回切就没确认修复",
			Judgment:  "业务恢复，根因入库",
			Cost:      13.0,
		},
	}
}

// Path 是一条完整的排障路径。
type Path struct {
	Name  string
	Steps []Step
}

// BestPath 是剧本原路：每步都有依据，不跳步。
func BestPath() Path {
	return Path{Name: "best", Steps: Script()}
}

// NaivePath 是常见的蛮干路径：等阈值告警、跳过任务层、凭单一证据下结论。
//
// 它多花的时间不是「查得慢」，而是「方向错」。这个对照比任何说教都直接。
func NaivePath() Path {
	return Path{Name: "naive", Steps: []Step{
		{
			ID: 1, Phase: PhaseDetect,
			Question:  "等告警",
			Look:      "阈值告警面板",
			Seen:      "全部指标在阈值内，没有一条报警；六分钟后业务方来问才有人看",
			Criterion: "阈值告警只能发现「超限」，发现不了「劣化」",
			Judgment:  "发现问题靠人报，不是靠系统",
			Cost:      6.0,
		},
		{
			ID: 2, Phase: PhaseNarrow,
			Question:  "全网逐台扫",
			Look:      "128 台交换机的全部端口计数器",
			Seen:      "扫到第 97 台才发现 leaf-7 上行口异常；前三十二台没过半",
			Criterion: "没有假设的扫描，是在几千个计数器里捞针",
			Judgment:  "找到了，但候选没被收敛过",
			Cost:      32.0,
		},
		{
			ID: 3, Phase: PhaseConfirm,
			Question:  "看到 CRC 涨就动手",
			Look:      "端口 CRC 计数",
			Seen:      "CRC 12 次/分钟，直接判定模块损坏，换了一件",
			Criterion: "单凭 CRC 不能定案：脏端面、弯折、连接器松动都会涨",
			Judgment:  "换错了，故障还在",
			Cost:      20.0,
		},
		{
			ID: 4, Phase: PhaseBound,
			Question:  "换完没好转，回头补课",
			Look:      "NCCL ring 组成与 GPU 侧指标",
			Seen:      "ring 正常；GPU 干净。前面六分钟的定界工作现在才做",
			Criterion: "定界应该在最前面做，它是把候选压一个数量级的唯一手段",
			Judgment:  "回到正确的路上，但已经过去了 58 分钟",
			Cost:      8.0,
		},
		{
			ID: 5, Phase: PhaseRecover,
			Question:  "总算轮到 drain",
			Look:      "drain 后的步耗时与回切",
			Seen:      "drain 后步耗时回到 390 ms；这次 lane 读数确认后才换模块",
			Criterion: "同第 10 步：回到基线 ±10% 才算恢复",
			Judgment:  "业务恢复，但换了两件模块",
			Cost:      13.0,
		},
	}}
}
