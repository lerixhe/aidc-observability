package main

// 四个场景的形态参数与注入命令。
//
// 形态参数（profile）描述「真机在观测窗口里会看到什么」；
// 注入命令（Step）描述「在容器拓扑里怎么造出这个现象」。
// 两者是分开的：真机版靠交换机 DCB 造 PFC，容器版靠队列与限速造等效观测形态。
// 详见 README 第 2 节的边界声明。

const (
	roleCulprit = "culprit"
	roleVictim  = "victim"
	roleHealthy = "healthy"
)

// Step 是一条注入动作：在哪台节点上、为什么、做什么。
type Step struct {
	Node string `json:"node"`
	Note string `json:"note"`
	Cmd  string `json:"cmd"`
}

// Scenario 是一个完整的可复现场景。
type Scenario struct {
	Name  string
	Kind  string // storm / deadlock / incast / hol
	Title string

	Window float64 // 观测窗口，秒
	Step   float64 // 采样粒度，秒
	Onset  float64 // 劣化起点，占窗口比例
	Seed   int64

	Ports     []profile
	Injection []Step
	Expect    []string // 判读要点，与第 07 课分诊表对齐
	Hardware  []string // 真机版的等价做法
}

// Scenarios 返回全部场景，顺序固定，便于 expected 稳定。
func Scenarios() []Scenario {
	return []Scenario{
		stormScene(),
		deadlockScene(),
		incastScene(),
		holScene(),
	}
}

// stormScene · PFC 风暴：不丢包，但一整片变慢。
func stormScene() Scenario {
	return Scenario{
		Name: "storm", Kind: "storm",
		Title:  "PFC 风暴：反压沿拓扑向上游蔓延，零丢包但区域性变慢",
		Window: 240, Step: 8, Onset: 0.25, Seed: 42,
		Ports: []profile{
			{
				Port: "leaf2:eth1", Role: roleCulprit,
				LatP50: 0.08, LatP99: 0.22, LatMul: 1.2,
				PauseBase: 0.1, PauseStep: 8, ECNBase: 0.1, ECNStep: 3.4,
				QBase: 8, QStep: 88, FECFrom: 1e-9, FECTo: 1e-9, Jitter: 0.04,
			},
			{
				Port: "leaf2:eth2", Role: roleVictim,
				LatP50: 0.08, LatP99: 0.22, LatMul: 5.2,
				PauseBase: 0.1, PauseStep: 6.8, ECNBase: 0.1, ECNStep: 2.6,
				QBase: 8, QStep: 72, FECFrom: 1e-9, FECTo: 1e-9, Jitter: 0.05,
			},
			{
				Port: "leaf1:eth3", Role: roleVictim,
				LatP50: 0.08, LatP99: 0.22, LatMul: 4.6,
				PauseBase: 0.1, PauseStep: 5.9, ECNBase: 0.1, ECNStep: 2.2,
				QBase: 8, QStep: 61, FECFrom: 1e-9, FECTo: 1e-9, Jitter: 0.05,
			},
			{
				Port: "leaf1:eth1", Role: roleHealthy,
				LatP50: 0.08, LatP99: 0.22, LatMul: 1,
				PauseBase: 0.1, ECNBase: 0.1, QBase: 8,
				FECFrom: 1e-9, FECTo: 1e-9, Jitter: 0.03,
			},
		},
		Injection: []Step{
			{"leaf2", "上行口做成大缓冲 + 低于需求的带宽：丢包被换成堆积，等效 PFC 的反压效果", "tc qdisc replace dev eth1 root tbf rate 40mbit burst 256kb latency 400ms"},
			{"h3", "接收端起服务", "iperf3 -s -1 -D"},
			{"h1", "两条大象流合起来超过上行带宽，队列开始常驻高位", "iperf3 -c 10.0.3.2 -t 240 -b 30M -l 64K -D"},
			{"h2", "第二条流，把过载推过阈值", "iperf3 -c 10.0.3.2 -t 240 -b 30M -l 64K -D"},
			{"leaf2", "看 backlog 与 drops：无 drops、backlog 常驻高位就是「丢包被换成堆积」", "tc -s qdisc show dev eth1"},
			{"leaf1", "上游侧看到的是出口排队引起的时延抬升", "tc -s qdisc show dev eth3"},
		},
		Expect: []string{
			"丢包率全程为 0，这是 PFC 家族的第一指纹",
			"pause 时长在多个端口同时抬到 5% 以上，且方向沿拓扑向上游扩散",
			"时延台阶式抬升后不回落到基线，说明根因没消除",
			"队列水线常驻 80% 以上，与 pause 同步而非滞后",
		},
		Hardware: []string{
			"上游交换机按优先级开 PFC，XOFF 阈值设 40–50% 缓冲",
			"`show priority-flow-control interface <intf> counters` 看 per-priority pause 帧数与时长",
			"把 pause 时长换算成占空比：10 分钟窗口里超过 10 秒即为严重",
		},
	}
}

// deadlockScene · PFC 死锁：不丢包，也不通。
func deadlockScene() Scenario {
	return Scenario{
		Name: "deadlock", Kind: "deadlock",
		Title:  "PFC 死锁：双向互等，报文卡在队列里永远出不去",
		Window: 120, Step: 4, Onset: 0.25, Seed: 42,
		Ports: []profile{
			{
				Port: "leaf2:eth1", Role: roleCulprit,
				LatP50: 0.08, LatP99: 0.22, LatMul: 0, // 时延超时
				PauseBase: 0.1, PauseStep: 100, ECNBase: 0.1, ECNStep: 0,
				QBase: 8, QStep: 92, FECFrom: 1e-9, FECTo: 1e-9, Jitter: 0.02,
			},
			{
				Port: "leaf2:eth2", Role: roleVictim,
				LatP50: 0.08, LatP99: 0.22, LatMul: 0,
				PauseBase: 0.1, PauseStep: 99, ECNBase: 0.1, ECNStep: 0,
				QBase: 8, QStep: 92, FECFrom: 1e-9, FECTo: 1e-9, Jitter: 0.02,
			},
			{
				Port: "leaf1:eth3", Role: roleVictim,
				LatP50: 0.08, LatP99: 0.22, LatMul: 0,
				PauseBase: 0.1, PauseStep: 98, ECNBase: 0.1, ECNStep: 0,
				QBase: 8, QStep: 88, FECFrom: 1e-9, FECTo: 1e-9, Jitter: 0.02,
			},
			{
				Port: "leaf1:eth1", Role: roleHealthy,
				LatP50: 0.08, LatP99: 0.22, LatMul: 1,
				PauseBase: 0.1, ECNBase: 0.1, QBase: 8,
				FECFrom: 1e-9, FECTo: 1e-9, Jitter: 0.03,
			},
		},
		Injection: []Step{
			{"leaf2", "上行限到极低带宽，逼出持久反压", "tc qdisc replace dev eth1 root tbf rate 2mbit burst 32kb latency 10s"},
			{"leaf1", "反方向也压，形成互相等待的环", "tc qdisc replace dev eth3 root tbf rate 2mbit burst 32kb latency 10s"},
			{"h3", "接收端起服务", "iperf3 -s -1 -D"},
			{"h1", "正向流与被压的反向流同时占着缓冲", "iperf3 -c 10.0.3.2 -t 120 -b 20M -l 64K -D || true"},
			{"h3", "反向流：谁也不让，谁也不走", "iperf3 -c 10.0.1.2 -t 120 -b 20M -l 64K -D || true"},
			{"leaf2", "看队列是否卡死在满水位、drops 是否为 0", "tc -s qdisc show dev eth1"},
			{"h1", "应用侧表现是全部超时，不是部分丢包", "ping -c 5 -W 2 10.0.3.2 || true"},
		},
		Expect: []string{
			"时延直接超时，报文根本没被丢，而是排队排到永远",
			"丢包率为 0 与超时并存，这是无损网络特有的矛盾组合",
			"pause 占空比顶到 90% 以上，ECN 归零（流已经不动了，没人再被标记）",
			"下游还没死，健康端口照常工作：故障范围由环决定，不是由流量决定",
		},
		Hardware: []string{
			"`show priority-flow-control watchdog` 确认死锁检测是否开启",
			"交换机上查 stuck 队列与永久 XOFF 的端口",
			"环检测：把「暂停方向」画成有向图，找强连通分量",
		},
	}
}

// incastScene · 多打一：周期性尖刺，与训练步同步。
func incastScene() Scenario {
	return Scenario{
		Name: "incast", Kind: "incast",
		Title:  "incast：多打一，汇聚口周期性冲高，丢包与 ECN 同时出现",
		Window: 40, Step: 0.5, Onset: 0.1, Seed: 42,
		Ports: []profile{
			{
				Port: "leaf2:eth2", Role: roleCulprit,
				LatP50: 0.08, LatP99: 0.22, LatMul: 2.6,
				PauseBase: 0.1, PauseStep: 28, ECNBase: 0.1, ECNStep: 4.8,
				LossBase: 0, LossStep: 1.4,
				QBase: 8, QStep: 88, Period: 4, Duty: 0.25,
				FECFrom: 1e-9, FECTo: 1e-9, Jitter: 0.06,
			},
			{
				Port: "leaf1:eth1", Role: roleVictim,
				LatP50: 0.08, LatP99: 0.22, LatMul: 1.9,
				PauseBase: 0.1, PauseStep: 9, ECNBase: 0.1, ECNStep: 2.1,
				LossBase: 0, LossStep: 0.3,
				QBase: 8, QStep: 34, Period: 4, Duty: 0.25,
				FECFrom: 1e-9, FECTo: 1e-9, Jitter: 0.06,
			},
			{
				Port: "leaf1:eth3", Role: roleVictim,
				LatP50: 0.08, LatP99: 0.22, LatMul: 2.2,
				PauseBase: 0.1, PauseStep: 12, ECNBase: 0.1, ECNStep: 2.6,
				LossBase: 0, LossStep: 0.2,
				QBase: 8, QStep: 41, Period: 4, Duty: 0.25,
				FECFrom: 1e-9, FECTo: 1e-9, Jitter: 0.06,
			},
			{
				Port: "leaf2:eth3", Role: roleHealthy,
				LatP50: 0.08, LatP99: 0.22, LatMul: 1,
				PauseBase: 0.1, ECNBase: 0.1, QBase: 8,
				FECFrom: 1e-9, FECTo: 1e-9, Jitter: 0.03,
			},
		},
		Injection: []Step{
			{"leaf2", "汇聚口限速：多打一时这里必然溢出", "tc qdisc replace dev eth2 root tbf rate 25mbit burst 32kb latency 20ms"},
			{"h3", "接收端起服务", "iperf3 -s -1 -D"},
			{"h1", "两个发送端同一时刻打同一个目的地", "iperf3 -c 10.0.3.2 -t 40 -b 20M -l 4K -D"},
			{"h2", "第二条汇聚流", "iperf3 -c 10.0.3.2 -t 40 -b 20M -l 4K -D"},
			{"leaf2", "看 dropped 与 backlog 的周期性：与 iperf3 的突发节奏同步", "tc -s qdisc show dev eth2"},
			{"h1", "从发送端看：重传与 RTT 的周期性尖刺", "ss -tin dst 10.0.3.2"},
		},
		Expect: []string{
			"丢包率非零，只在尖刺窗口里非零，这是与 PFC 家族的分水岭",
			"队列与 ECN 尖刺的周期等于训练步的周期，可用日志对齐验证",
			"受影响路径集合随时间稳定，指向少数汇聚点，不是成片蔓延",
			"汇聚点上游的全部发送端都被波及，但各自的问题程度不同",
		},
		Hardware: []string{
			"汇聚设备上按队列看 ECN 标记率、PFC 触发次数、discard",
			"与训练 step 时间线对齐：能对齐的是 incast，不能的是风暴",
			"调小上行突发：`-l 4K` 改小、增加并发数，观察尖刺是否变陡",
		},
	}
}

// holScene · 队头阻塞：无辜路径陪绑。
func holScene() Scenario {
	return Scenario{
		Name: "hol", Kind: "hol",
		Title:  "队头阻塞：共享瓶颈上的无辜流被大象流拖慢",
		Window: 240, Step: 8, Onset: 0.1, Seed: 42,
		Ports: []profile{
			{
				Port: "leaf2:eth1", Role: roleCulprit,
				LatP50: 0.08, LatP99: 0.22, LatMul: 1.3,
				PauseBase: 0.1, PauseStep: 2.9, ECNBase: 0.1, ECNStep: 0.8,
				QBase: 8, QStep: 68, FECFrom: 1e-9, FECTo: 1e-9, Jitter: 0.05,
			},
			{
				Port: "leaf2:eth3", Role: roleVictim,
				LatP50: 0.08, LatP99: 0.22, LatMul: 1.9,
				PauseBase: 0.1, PauseStep: 1.4, ECNBase: 0.1, ECNStep: 0.3,
				QBase: 8, QStep: 22, FECFrom: 1e-9, FECTo: 1e-9, Jitter: 0.05,
			},
			{
				Port: "leaf2:eth2", Role: roleVictim,
				LatP50: 0.08, LatP99: 0.22, LatMul: 2.1,
				PauseBase: 0.1, PauseStep: 1.1, ECNBase: 0.1, ECNStep: 0.2,
				QBase: 8, QStep: 18, FECFrom: 1e-9, FECTo: 1e-9, Jitter: 0.05,
			},
			{
				Port: "leaf1:eth1", Role: roleHealthy,
				LatP50: 0.08, LatP99: 0.22, LatMul: 1,
				PauseBase: 0.1, ECNBase: 0.1, QBase: 8,
				FECFrom: 1e-9, FECTo: 1e-9, Jitter: 0.03,
			},
		},
		Injection: []Step{
			{"leaf2", "共享上行做成瓶颈，队列只有一个", "tc qdisc replace dev eth1 root tbf rate 40mbit burst 128kb latency 200ms"},
			{"h3", "慢目的地起服务", "iperf3 -s -1 -D"},
			{"h4", "健康目的地起服务", "iperf3 -s -1 -D"},
			{"h1", "肇事大象流：吃满共享队列", "iperf3 -c 10.0.3.2 -t 240 -b 38M -l 64K -D"},
			{"h2", "无辜小流：目的地健康，却和肇事流排同一个队列", "iperf3 -c 10.0.4.2 -t 240 -b 1M -l 4K"},
			{"leaf2", "看共享口的 backlog：两条流的水都在这里", "tc -s qdisc show dev eth1"},
			{"h2", "无辜流的 RTT 与重传，是它「清白」的证据", "ss -tin dst 10.0.4.2"},
		},
		Expect: []string{
			"丢包率接近 0，时延只是温和抬升（1.5–2.5 倍），不像风暴那样成片",
			"受害端口与肇事端口共享同一段，其余各段全部正常",
			"pause 与 ECN 都低：机制没有失效，只是队列在排队",
			"设备级均值看不出问题，只有路径级数据能证明「这条路径除了共享段以外都干净」",
		},
		Hardware: []string{
			"端口上开 INT，看逐跳时延：集中在共享段那一跳",
			"按流的路径做时间相关性分析，确认受害流与肇事流同时进出同一队列",
			"考虑 per-flow 队列或更大 hash 熵，减少共享",
		},
	}
}

// FindScenario 按名字查场景，空名字回退到第一个。
func FindScenario(name string) (Scenario, bool) {
	for _, s := range Scenarios() {
		if s.Name == name || s.Kind == name {
			return s, true
		}
	}
	return Scenario{}, false
}
