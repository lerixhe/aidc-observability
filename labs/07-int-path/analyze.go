package main

import "fmt"

// 判读阈值。教学模型用固定值，真机上要按端口缓冲与基线重定。
const (
	TxBusyBP    = 8500 // 出口利用率 ≥ 85% 视为接近打满
	QueueHighKB = 500  // 出口队列 ≥ 500 KB 视为堆积（按 1000 KB 端口缓冲口径）
	HopSlowNs   = 20000
)

// Verdict 是一跳的归因结论。
type Verdict string

const (
	VerdictOK          Verdict = "ok"
	VerdictEgress      Verdict = "本跳出口拥塞"
	VerdictUpstream    Verdict = "上游争抢"
	VerdictForwarding  Verdict = "转发面偏慢"
	VerdictUnclearNote Verdict = "无法归因"
)

// HopAnalysis 是单跳的判读结果。
type HopAnalysis struct {
	Index   int
	Meta    HopMeta
	SharePc float64 // 该跳时延占全程比例，百分比
	Verdict Verdict
	Reason  string
}

// Analysis 是整条路径的判读结果。
type Analysis struct {
	FlowID     uint32
	TotalNs    uint32
	Hops       []HopAnalysis
	WorstIndex int
}

// ClassifyHop 把「利用率 × 队列 × 停留时长」三个量组合成归因结论。
//
// 关键判断在课 12：hop latency 高不等于本跳有问题，要把队列与出口利用率一起看。
func ClassifyHop(h HopMeta) (Verdict, string) {
	busy := h.TxUtilBP >= TxBusyBP
	congested := h.QueueKB >= QueueHighKB
	slow := h.HopLatNs >= HopSlowNs

	switch {
	case slow && busy && congested:
		return VerdictEgress, "出口接近打满且队列堆积，是这一跳的出口在排队"
	case slow && congested && !busy:
		return VerdictUpstream, "队列堆着但出口没满，多股流在抢同一出口，问题在上游分布"
	case slow && !congested && !busy:
		return VerdictForwarding, "队列空、出口闲而停留久，开销在转发面（查表、ACL、微码）"
	case slow:
		return VerdictUnclearNote, "停留偏长但队列与利用率都不典型，需要更多证据"
	default:
		return VerdictOK, "—"
	}
}

// Analyze 做逐跳分解。输入是解码后的元数据，输出带比例与归因。
func Analyze(flowID uint32, hops []HopMeta) Analysis {
	var total uint32
	for _, h := range hops {
		total += h.HopLatNs
	}

	a := Analysis{FlowID: flowID, TotalNs: total, WorstIndex: -1}
	var worst uint32
	for i, h := range hops {
		v, reason := ClassifyHop(h)
		share := 0.0
		if total > 0 {
			share = float64(h.HopLatNs) / float64(total) * 100
		}
		a.Hops = append(a.Hops, HopAnalysis{Index: i + 1, Meta: h, SharePc: share, Verdict: v, Reason: reason})
		if h.HopLatNs > worst {
			worst = h.HopLatNs
			a.WorstIndex = i
		}
	}
	return a
}

// Suspects 返回所有非 ok 的跳，按时延降序。
func (a Analysis) Suspects() []HopAnalysis {
	out := make([]HopAnalysis, 0, len(a.Hops))
	for _, h := range a.Hops {
		if h.Verdict != VerdictOK {
			out = append(out, h)
		}
	}
	// 简单的插入排序，跳数很少
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].Meta.HopLatNs > out[j-1].Meta.HopLatNs; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// Headline 给出一句话结论，报告与测试都用它。
func (a Analysis) Headline() string {
	s := a.Suspects()
	switch len(s) {
	case 0:
		return "全跳健康：没有哪一跳同时满足「时延高 + 队列或利用率异常」"
	case 1:
		h := s[0]
		return fmt.Sprintf("定位到第 %d 跳（%s）：%s", h.Index, h.Meta.Name(), h.Verdict)
	default:
		h := s[0]
		return fmt.Sprintf("多跳可疑，最重的是第 %d 跳（%s）：%s，另有 %d 跳待查",
			h.Index, h.Meta.Name(), h.Verdict, len(s)-1)
	}
}
