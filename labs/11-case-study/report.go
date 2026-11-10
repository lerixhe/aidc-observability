package main

import (
	"fmt"
	"io"
	"strings"
)

// 输出格式固定、数字格式固定，锁进 expected/ 供 CI 比对。
//
// 表格对齐要处理中文宽度：一个汉字占两列，用 len() 数会错位。

// dispWidth 算显示宽度：CJK 与全角标点按两列算。
func dispWidth(s string) int {
	w := 0
	for _, r := range s {
		if isWide(r) {
			w += 2
		} else {
			w++
		}
	}
	return w
}

func isWide(r rune) bool {
	switch {
	case r >= 0x1100 && r <= 0x115F, // 谚文字母
		r >= 0x2E80 && r <= 0xA4CF, // CJK 部首、汉字、假名
		r >= 0xAC00 && r <= 0xD7A3, // 谚文音节
		r >= 0xF900 && r <= 0xFAFF, // CJK 兼容汉字
		r >= 0xFE30 && r <= 0xFE6F, // CJK 兼容形式
		r >= 0xFF00 && r <= 0xFF60, // 全角 ASCII
		r >= 0xFFE0 && r <= 0xFFE6:
		return true
	}
	return false
}

// pad 按显示宽度右补空格；超宽则截断并加省略号。
func pad(s string, width int) string {
	w := dispWidth(s)
	if w == width {
		return s
	}
	if w < width {
		return s + strings.Repeat(" ", width-w)
	}
	var b strings.Builder
	cur := 0
	for _, r := range s {
		rw := 1
		if isWide(r) {
			rw = 2
		}
		if cur+rw > width-1 {
			break
		}
		b.WriteRune(r)
		cur += rw
	}
	b.WriteString("…")
	out := b.String()
	for dispWidth(out) < width {
		out += " "
	}
	return out
}

// ReportScript 打印完整剧本：十步，每步四个要素。
func ReportScript(w io.Writer, steps []Step) {
	fmt.Fprintln(w, "== 剧本：十步，四把刀 ==")
	fmt.Fprintln(w, "任务层 → 主机层 → 网络层 → 光层，每一层只做一件事：把候选压一个数量级。")
	fmt.Fprintln(w)
	for _, s := range steps {
		fmt.Fprintf(w, "[%2d] %s · %s（%.1f 分钟）\n", s.ID, s.Phase, s.Question, s.Cost)
		fmt.Fprintf(w, "     看：%s\n", s.Look)
		fmt.Fprintf(w, "     见：%s\n", s.Seen)
		fmt.Fprintf(w, "     判：%s → %s\n", s.Criterion, s.Judgment)
		fmt.Fprintln(w)
	}
}

// ReportRun 打印一条路径的执行过程。
//
// 一步三行而不是一行五列：中文表格一挤就截断，截断掉的恰好是读数，
// 而读数正是这套剧本要教的东西。
func ReportRun(w io.Writer, r Run) {
	fmt.Fprintf(w, "== 路径：%s（%d 步，合计 %s 分钟）==\n", r.Path.Name, r.Units, fmtMin(r.Total))
	for _, s := range r.Path.Steps {
		fmt.Fprintf(w, "[%2d] %s · %s · %s 分钟\n", s.ID, s.Phase, s.Question, fmtMin(s.Cost))
		fmt.Fprintf(w, "     见：%s\n", s.Seen)
		fmt.Fprintf(w, "     判：%s\n", s.Judgment)
	}
	fmt.Fprintln(w)
}

// ReportMTTR 打印 MTTR 五段分解。
func ReportMTTR(w io.Writer, r Run) {
	fmt.Fprintf(w, "== MTTR 分解：%s 分钟 ==\n", fmtMin(r.Total))
	fmt.Fprintln(w, "阶段    分钟   占比   步骤")
	for _, pc := range r.Phases {
		if pc.Minutes == 0 {
			continue
		}
		var ids []string
		for _, id := range pc.Steps {
			ids = append(ids, fmt.Sprintf("%d", id))
		}
		fmt.Fprintf(w, "%-6s  %-5s  %5.1f%%  %s\n",
			pad(string(pc.Phase), 6), fmtMin(pc.Minutes), r.Share(pc.Phase)*100,
			strings.Join(ids, "、"))
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "读法：")
	fmt.Fprintln(w, "  1. 定界与缩小通常是大头：候选从几百个收敛到一处，全靠这两段")
	fmt.Fprintln(w, "  2. 「发现」一段能压到几分钟，靠的是路径级探测，不是设备级均值")
	fmt.Fprintln(w, "  3. 「恢复」一段压不下去：drain、观察、换件、回切，都要物理时间")
	fmt.Fprintln(w)
}

// ReportCompare 打印两条路径的对照。这是本 lab 的结论所在。
func ReportCompare(w io.Writer, best, naive Run) {
	fmt.Fprintln(w, "== 对照：同一份数据，两条路径 ==")
	fmt.Fprintln(w, "路径    发现   定界   缩小   确认   恢复   合计   结论")
	for _, r := range []Run{best, naive} {
		fmt.Fprintf(w, "%-6s  %s  %s  %s  %s  %s  %s  %s\n",
			pad(r.Path.Name, 6),
			pad(fmtMin(r.Minutes(PhaseDetect)), 5),
			pad(fmtMin(r.Minutes(PhaseBound)), 5),
			pad(fmtMin(r.Minutes(PhaseNarrow)), 5),
			pad(fmtMin(r.Minutes(PhaseConfirm)), 5),
			pad(fmtMin(r.Minutes(PhaseRecover)), 5),
			pad(fmtMin(r.Total), 5),
			r.Verdict())
	}
	delta := naive.Total - best.Total
	fmt.Fprintln(w)
	fmt.Fprintf(w, "差距 %s 分钟，接近两倍。多出来的时间不在「查得慢」，在「方向错」：\n", fmtMin(delta))
	fmt.Fprintln(w, "  - 等阈值告警：所有指标都在阈值内，这条路径天然发现不了劣化")
	fmt.Fprintln(w, "  - 跳过任务层与主机层：定界没做，候选从几百个开始捞")
	fmt.Fprintln(w, "  - 凭 CRC 单证据换件：换错一件，白花 20 分钟")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "三条原则对应三次教训：先看趋势再看阈值、先定界再缩小、证据要来自不同来源。")
	fmt.Fprintln(w)
}

// Minutes 返回某阶段的分钟数。
func (r Run) Minutes(p Phase) float64 {
	for _, pc := range r.Phases {
		if pc.Phase == p {
			return pc.Minutes
		}
	}
	return 0
}
