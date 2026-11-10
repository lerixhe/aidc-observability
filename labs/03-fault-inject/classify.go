package main

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// 判读器：把一段遥测样本（多个端口、一个时间窗口）压成特征，再按第 07 课的
// 分诊逻辑给出结论。这是本 lab 最值得读的一段代码 —— 它的判据就是生产里
// 靠经验攒出来的那几条，只是写成了可执行的形式。
//
// 阈值的取值依据（与第 07、21 课一致）：
//
//	pause 占空比   10 分钟窗口 > 10 秒即为严重，换算成占比约 1.7%；
//	               这里取 5% 作为「风暴级」的判定线，留了安全余量
//	丢包率         无损网络里任何持续丢包都是异常，判定线取 0.05%
//	ECN 标记率     跑满带宽时 1–2% 常见，> 3% 持续存在即异常
//	时延倍数       1.2 倍以内算噪声（探测本身有抖动），超过即视为劣化

// Thresholds 集中放判读阈值，便于读者改参数做敏感度实验。
type Thresholds struct {
	LossPct      float64 // 丢包率判定线，%
	PauseSevere  float64 // pause 占空比的严重线，%
	LatRatio     float64 // 时延劣化判定线，倍
	TimeoutFrac  float64 // 超时占比判定线
	PeriodCV     float64 // 周期稳定性的变异系数上限
	MinPeriodPts int     // 判定周期性所需的最少尖刺个数
}

// DefaultThresholds 给出默认判读口径。
func DefaultThresholds() Thresholds {
	return Thresholds{
		LossPct:      0.05,
		PauseSevere:  5,
		LatRatio:     1.2,
		TimeoutFrac:  0.15,
		PeriodCV:     0.5,
		MinPeriodPts: 3,
	}
}

// Features 是从一份样本里算出来的判读依据，打印在结论前面。
type Features struct {
	Ports        int
	TimeoutFrac  float64
	LossMax      float64
	LossMean     float64
	PauseMax     float64
	PauseMean    float64
	ECNMax       float64
	QMax         float64
	LatRatio     float64
	Periodic     bool
	Period       float64
	Affected     int      // 出现劣化的端口数
	AffectedList []string // 劣化端口清单
}

// Verdict 是判读结论。
type Verdict struct {
	Kind       string // storm / deadlock / incast / hol / healthy / lossy
	Confidence string // 高 / 中 / 低
	Features   Features
	Evidence   []string
	Next       []string
}

// byPort 把样本按端口切开，保持出现顺序。
func byPort(rows []Row) (order []string, groups map[string][]Row) {
	groups = make(map[string][]Row)
	for _, r := range rows {
		if _, ok := groups[r.Port]; !ok {
			order = append(order, r.Port)
		}
		groups[r.Port] = append(groups[r.Port], r)
	}
	return order, groups
}

// Extract 计算判读特征。baseline 取每个端口窗口最前面 10% 的采样均值，
// 因为观测窗口的头部通常还处在故障前状态。
func Extract(rows []Row, th Thresholds) Features {
	order, groups := byPort(rows)
	f := Features{Ports: len(order)}
	if len(order) == 0 {
		return f
	}

	var lossAll, pauseAll []float64
	timeouts, total := 0, 0
	maxRatio := 1.0

	for _, port := range order {
		g := groups[port]
		head := len(g) / 10
		if head < 1 {
			head = 1
		}
		var baseP99 float64
		for _, r := range g[:head] {
			if r.LatP99 > 0 {
				baseP99 += r.LatP99
			}
		}
		baseP99 /= float64(head)

		var pLoss, pPause float64
		portDegraded := false
		for _, r := range g {
			total++
			if r.LatP99 < 0 {
				timeouts++
			} else if baseP99 > 0 && r.LatP99/baseP99 > maxRatio {
				maxRatio = r.LatP99 / baseP99
			}
			if r.LossPct > pLoss {
				pLoss = r.LossPct
			}
			if r.PausePct > pPause {
				pPause = r.PausePct
			}
			if r.ECNPct > f.ECNMax {
				f.ECNMax = r.ECNPct
			}
			if r.QDepthKB > f.QMax {
				f.QMax = r.QDepthKB
			}
			if r.Degraded() {
				portDegraded = true
			}
			if baseP99 > 0 && r.LatP99 > 0 && r.LatP99/baseP99 > th.LatRatio {
				portDegraded = true
			}
			lossAll = append(lossAll, r.LossPct)
			pauseAll = append(pauseAll, r.PausePct)
		}
		if pLoss > f.LossMax {
			f.LossMax = pLoss
		}
		if pPause > f.PauseMax {
			f.PauseMax = pPause
		}
		if portDegraded {
			f.Affected++
			f.AffectedList = append(f.AffectedList, port)
		}
	}

	if total > 0 {
		f.TimeoutFrac = float64(timeouts) / float64(total)
	}
	f.LossMean = mean(lossAll)
	f.PauseMean = mean(pauseAll)
	f.LatRatio = maxRatio

	// 周期性只在「尖刺型」样本上有意义：没有丢包也没有 pause 尖刺时，
	// 算出来的周期是噪声。
	spikes := spikeTimes(rows, groups, order, th)
	f.Periodic, f.Period = periodicity(spikes, th)
	return f
}

// spikeTimes 找出尖刺的上升沿，用于周期性判定。
// 取上升沿而不是「超阈值的点」：持续劣化只有一个上升沿，周期突发有很多个，
// 两种形态的区别就出来了。
func spikeTimes(rows []Row, groups map[string][]Row, order []string, th Thresholds) []float64 {
	step := sampleStep(groups, order)
	var ts []float64
	for _, port := range order {
		prev := false
		for _, r := range groups[port] {
			cur := r.LossPct > th.LossPct || r.PausePct > th.PauseSevere
			if cur && !prev {
				ts = append(ts, r.T)
			}
			prev = cur
		}
	}
	sort.Float64s(ts)

	// 同一轮尖刺在不同端口上的上升沿可能差一个采样点，合并到半个步长以内
	mergeGap := step * 0.5
	if mergeGap <= 0 {
		mergeGap = 0.25
	}
	var merged []float64
	for _, t := range ts {
		if len(merged) == 0 || t-merged[len(merged)-1] > mergeGap {
			merged = append(merged, t)
		}
	}
	return merged
}

// sampleStep 从样本里反推采样步长，避免判读器依赖外部参数。
func sampleStep(groups map[string][]Row, order []string) float64 {
	for _, port := range order {
		g := groups[port]
		if len(g) >= 2 && g[1].T > g[0].T {
			return g[1].T - g[0].T
		}
	}
	return 0
}

// periodicity 用相邻间隔的变异系数判断是否周期性。
func periodicity(ts []float64, th Thresholds) (bool, float64) {
	if len(ts) < th.MinPeriodPts+1 {
		return false, 0
	}
	var gaps []float64
	for i := 1; i < len(ts); i++ {
		gaps = append(gaps, ts[i]-ts[i-1])
	}
	m := mean(gaps)
	if m <= 0 {
		return false, 0
	}
	var sq float64
	for _, g := range gaps {
		sq += (g - m) * (g - m)
	}
	cv := math.Sqrt(sq/float64(len(gaps))) / m
	return cv < th.PeriodCV, m
}

// Classify 按分诊顺序给出结论。顺序本身是知识：
// 先判有没有丢包，再判丢包是持续还是周期，再判 pause 强度，最后才轮到时延。
func Classify(rows []Row, th Thresholds) Verdict {
	f := Extract(rows, th)
	v := Verdict{Features: f}

	switch {
	case f.Ports == 0:
		v.Kind, v.Confidence = "empty", "低"
		v.Evidence = []string{"样本为空，先确认采集是否跑通"}
		return v

	case f.TimeoutFrac > th.TimeoutFrac && f.LossMax < th.LossPct:
		v.Kind, v.Confidence = "deadlock", "高"
		v.Evidence = []string{
			fmt.Sprintf("超时样本占比 %.0f%%，但丢包率峰值只有 %.2f%%", f.TimeoutFrac*100, f.LossMax),
			fmt.Sprintf("pause 占空比峰值 %.0f%%，队列顶到 %.0f KB", f.PauseMax, f.QMax),
			"报文没被丢，而是排队排到出不去 —— 无损网络特有的矛盾组合",
		}
		v.Next = []string{
			"查 PFC watchdog 是否开启，有无 stuck 队列事件",
			"把暂停方向画成有向图，找强连通分量定位环",
			"先做端口隔离止血，再改 PFC 水线与拓扑",
		}

	case f.LossMax > th.LossPct && f.Periodic:
		v.Kind, v.Confidence = "incast", "高"
		v.Evidence = []string{
			fmt.Sprintf("丢包率峰值 %.2f%%，出现在周期性尖刺里，间隔约 %.1f 秒", f.LossMax, f.Period),
			fmt.Sprintf("ECN 标记率峰值 %.1f%%，队列峰值 %.0f KB", f.ECNMax, f.QMax),
			fmt.Sprintf("劣化端口 %d/%d，方向指向汇聚点而非成片蔓延", f.Affected, f.Ports),
		}
		v.Next = []string{
			"拿尖刺时间点与训练 step 日志对齐，确认是同步突发",
			"查汇聚设备的 ECN / PFC / discard 三级指标，确认机制是否在工作",
			"考虑增大汇聚口缓冲、打开 ECN 并把 Kmin 调低",
		}

	case f.LossMax > th.LossPct:
		v.Kind, v.Confidence = "lossy", "中"
		v.Evidence = []string{
			fmt.Sprintf("丢包率峰值 %.2f%%，但不是周期性尖刺", f.LossMax),
			fmt.Sprintf("pause 占空比峰值 %.0f%%：反压没接管，丢包直接发生", f.PauseMax),
		}
		v.Next = []string{
			"先确认 PFC 是否真的开了：配置、优先级映射、两端一致性",
			"再确认 ECN 水线是否过高，导致标记不足",
		}

	case f.PauseMax > th.PauseSevere && f.LatRatio > 2.5:
		v.Kind, v.Confidence = "storm", "高"
		v.Evidence = []string{
			fmt.Sprintf("丢包率峰值 %.2f%%：全程不丢包", f.LossMax),
			fmt.Sprintf("pause 占空比峰值 %.0f%%，平均 %.1f%%", f.PauseMax, f.PauseMean),
			fmt.Sprintf("时延抬到基线 %.1f 倍，劣化端口 %d/%d，成片蔓延", f.LatRatio, f.Affected, f.Ports),
		}
		v.Next = []string{
			"逐跳拉 per-priority pause 计数，找第一个「pause 涨但入流量正常」的设备",
			"核对这几跳的 PFC 水线与优先级配置是否一致",
			"确认源头那个消费不动的端口，而不是拥塞端口",
		}

	case f.LatRatio > th.LatRatio:
		v.Kind, v.Confidence = "hol", "中"
		v.Evidence = []string{
			fmt.Sprintf("时延温和抬升到基线 %.1f 倍，丢包率峰值 %.2f%%", f.LatRatio, f.LossMax),
			fmt.Sprintf("pause 与 ECN 都低（%.1f%% / %.1f%%）：机制没失效，只是队列在排队", f.PauseMax, f.ECNMax),
			fmt.Sprintf("劣化端口 %d/%d，集中在共享段", f.Affected, f.Ports),
		}
		v.Next = []string{
			"用 INT 看逐跳时延，把怀疑范围压到共享的那一跳",
			"做流间时间相关性：受害流与肇事流是否同时进出同一队列",
			"评估 per-flow 队列或更大 hash 熵，减少无关流共享",
		}

	default:
		v.Kind, v.Confidence = "healthy", "中"
		v.Evidence = []string{
			fmt.Sprintf("丢包 %.2f%%、pause %.1f%%、时延 %.1f 倍，三项都在噪声范围内", f.LossMax, f.PauseMax, f.LatRatio),
		}
		v.Next = []string{"换个时间窗口或换个端口集合再看：可能故障不在这片区域"}
	}
	return v
}

func mean(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	var s float64
	for _, x := range xs {
		s += x
	}
	return s / float64(len(xs))
}

// KindTitle 给结论一个中文名，输出时用。
func KindTitle(kind string) string {
	switch kind {
	case "storm":
		return "PFC 风暴"
	case "deadlock":
		return "PFC 死锁"
	case "incast":
		return "incast（多打一）"
	case "hol":
		return "队头阻塞 HoL"
	case "lossy":
		return "持续性丢包（PFC 未生效或水线过高）"
	case "healthy":
		return "未见异常"
	case "empty":
		return "样本为空"
	}
	return kind
}

// ExplainMissing 检查判读结论与场景预期是否一致，供 -verify 使用。
func ExplainMissing(got, want string) string {
	if got == want {
		return ""
	}
	return strings.Join([]string{
		"判读结论与场景不符：场景是「" + KindTitle(want) + "」，判读给了「" + KindTitle(got) + "」",
		"通常是注入没生效（qdisc 没挂上、流量没打满），或观测窗口太短",
	}, "；")
}
