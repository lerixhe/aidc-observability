package main

import (
	"fmt"
	"io"
	"strings"
)

// Report 打印整份实验报告。
func Report(w io.Writer, ps []Path, an []Analysis, rows []SampleRow, pktRate, burstUs int) {
	fmt.Fprintln(w, "== 07-int-path · INT 逐跳元数据编码与判读 ==")
	fmt.Fprintf(w, "路径 %d 跳 · 每跳 %d 字节 · 载荷 %d 字节 · 采样对照 %d 档\n",
		len(ps[0].Hops), HopSize, WireBytes(len(ps[0].Hops)), len(rows))
	fmt.Fprintf(w, "微突发假设 %d 报文/秒 · 单次 %d µs · 一次突发 %d 个报文\n\n",
		pktRate, burstUs, BurstPkts(pktRate, burstUs))

	// 1. 编解码往返
	fmt.Fprintln(w, "编解码往返校验")
	fmt.Fprintln(w, "------------------------------------------------------------")
	ok := true
	for _, p := range ps {
		back, err := Decode(p.Payload())
		if err != nil {
			ok = false
			fmt.Fprintf(w, "  %-12s ❌ 解码失败：%v\n", p.Scenario, err)
			continue
		}
		same := len(back) == len(p.Hops)
		for i := range back {
			if back[i] != p.Hops[i] {
				same = false
			}
		}
		if same {
			fmt.Fprintf(w, "  %-12s ✅ %d 跳，%d 字节，往返一致\n", p.Scenario, len(back), len(p.Payload()))
		} else {
			ok = false
			fmt.Fprintf(w, "  %-12s ❌ 往返字段不一致\n", p.Scenario)
		}
	}
	// 截断载荷必须报错，否则解析会把残字节当垃圾读
	if _, err := Decode(Encode(ps[0].Hops)[:HopSize+5]); err == nil {
		ok = false
		fmt.Fprintln(w, "  truncated     ❌ 截断载荷没有报错")
	} else {
		fmt.Fprintln(w, "  truncated     ✅ 截断载荷被拒绝，不会读出垃圾字段")
	}
	_ = ok

	// 2. 逐跳判读
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "逐跳判读")
	fmt.Fprintln(w, "------------------------------------------------------------")
	for i, a := range an {
		fmt.Fprintf(w, "%s\n", a.Headline())
		fmt.Fprintf(w, "  场景 %-12s 全程 %6.1f µs\n", ps[i].Scenario, float64(a.TotalNs)/1000)
		fmt.Fprintf(w, "  %-3s %-8s %-4s %-4s %10s %8s %8s %8s  %-14s %s\n",
			"#", "设备", "入", "出", "停留(µs)", "队列KB", "利用%", "占比", "归因", "依据")
		for _, h := range a.Hops {
			mark := " "
			if h.Verdict != VerdictOK {
				mark = "*"
			}
			fmt.Fprintf(w, "  %s%-2d %-8s %-4d %-4d %10.1f %8d %8.1f %8s  %-14s %s\n",
				mark, h.Index, h.Meta.Name(), h.Meta.Ingress, h.Meta.Egress,
				float64(h.Meta.HopLatNs)/1000, h.Meta.QueueKB, h.Meta.TxUtilPct(),
				fmt.Sprintf("%.1f%%", h.SharePc), h.Verdict, h.Reason)
		}
		fmt.Fprintln(w, "")
	}

	// 3. 采样对照
	fmt.Fprintln(w, "采样率 vs 微突发捕获")
	fmt.Fprintln(w, "------------------------------------------------------------")
	fmt.Fprintf(w, "  %-8s %-8s %-10s %-10s %s\n", "采样", "突发报文", "解析捕获", "实测捕获", "连续 60 次全错过")
	for _, r := range rows {
		fmt.Fprintf(w, "  %-8s %-8d %-10s %-10s %s\n",
			fmt.Sprintf("1:%d", r.Rate), r.BurstPkts,
			fmt.Sprintf("%.1f%%", r.Analytic*100), fmt.Sprintf("%.1f%%", r.Measured*100),
			fmt.Sprintf("%.1f%%", MissStreak(r.Analytic, 60)*100))
	}
	fmt.Fprintf(w, "  %s\n", SamplingSummary(rows))
	fmt.Fprintf(w, "  元数据开销：5 跳 %d 字节，相对 1450 字节报文 %.1f%%\n", WireBytes(5), OverheadPct(1450, 5))
}

// CSV 输出逐跳明细，便于画图。
func CSV(a Analysis) string {
	var b strings.Builder
	b.WriteString("flow,hop,switch,ingress,egress,hop_lat_us,queue_kb,tx_util_pct,share_pct,verdict\n")
	for _, h := range a.Hops {
		fmt.Fprintf(&b, "%08x,%d,%s,%d,%d,%.3f,%d,%.2f,%.2f,%s\n",
			a.FlowID, h.Index, h.Meta.Name(), h.Meta.Ingress, h.Meta.Egress,
			float64(h.Meta.HopLatNs)/1000, h.Meta.QueueKB, h.Meta.TxUtilPct(), h.SharePc, h.Verdict)
	}
	return b.String()
}
