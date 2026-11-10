package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// 输出全部锁定在 expected/ 里：格式固定、顺序固定、数字格式固定，
// 这样模型一改，diff 会自己跳出来。

// WriteSampleCSV 把模拟样本落成 CSV，供分类器读取，也供读者用任意工具画图。
func WriteSampleCSV(w io.Writer, rows []Row) error {
	var b strings.Builder
	b.WriteString("port,role,t_s,lat_p50_ms,lat_p99_ms,loss_pct,pause_pct,ecn_pct,qdepth_kb,crc_per_min,fec_ber\n")
	for _, r := range rows {
		fmt.Fprintf(&b, "%s,%s,%.1f,%.2f,%.2f,%.3f,%.2f,%.2f,%.1f,%.1f,%.1e\n",
			r.Port, r.Role, r.T, r.LatP50, r.LatP99, r.LossPct, r.PausePct, r.ECNPct,
			r.QDepthKB, r.CRCPerMin, r.FECBER)
	}
	_, err := io.WriteString(w, b.String())
	return err
}

// SaveSample 把样本写到目录下，文件名固定为 model-<场景>.csv。
func SaveSample(dir string, s Scenario, rows []Row) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "model-"+s.Name+".csv")
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if err := WriteSampleCSV(f, rows); err != nil {
		return "", err
	}
	return path, nil
}

// ReportPlan 打印注入计划。dry-run 时这段输出就是可照着敲的剧本。
func ReportPlan(w io.Writer, s Scenario) {
	fmt.Fprintf(w, "== %s · 注入计划 ==\n", s.Name)
	fmt.Fprintf(w, "%s\n\n", s.Title)
	fmt.Fprintf(w, "窗口 %.0f 秒 · 采样 %.1f 秒 · 劣化起点 %.0f%% 处（第 %.0f 秒）\n\n",
		s.Window, s.Step, s.Onset*100, s.Onset*s.Window)

	fmt.Fprintln(w, "拓扑：h1 h2 ── leaf1 ── spine1 ── leaf2 ── h3(慢) h4(健康)")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "次序  节点     动作                                                  命令")
	fmt.Fprintln(w, "----  -------  ----------------------------------------------------  ----")
	for i, st := range s.Injection {
		fmt.Fprintf(w, "%2d    %-7s  %-52s  %s\n", i+1, st.Node, st.Note, st.Cmd)
	}
	fmt.Fprintln(w)

	fmt.Fprintln(w, "判读要点（真机上该看到什么）：")
	for _, e := range s.Expect {
		fmt.Fprintf(w, "  - %s\n", e)
	}
	fmt.Fprintln(w)

	fmt.Fprintln(w, "真机等价做法（容器里造不出来的部分）：")
	for _, h := range s.Hardware {
		fmt.Fprintf(w, "  - %s\n", h)
	}
	fmt.Fprintln(w)

	fmt.Fprintln(w, "清理：make destroy   # 或 docker exec <节点> tc qdisc del dev <if> root")
	fmt.Fprintln(w)
}

// portStat 是单个端口在窗口内的汇总，用来看形态。
type portStat struct {
	Port     string
	Role     string
	BaseP99  float64
	PeakP99  float64
	LossMax  float64
	PauseMax float64
	ECNMax   float64
	QMax     float64
	Timeout  int
}

// summarize 按端口聚合样本。
func summarize(rows []Row) []portStat {
	order, groups := byPort(rows)
	out := make([]portStat, 0, len(order))
	for _, port := range order {
		g := groups[port]
		st := portStat{Port: port, Role: g[0].Role}
		head := len(g) / 10
		if head < 1 {
			head = 1
		}
		for _, r := range g[:head] {
			if r.LatP99 > st.BaseP99 {
				st.BaseP99 = r.LatP99
			}
		}
		for _, r := range g {
			if r.LatP99 < 0 {
				st.Timeout++
			} else if r.LatP99 > st.PeakP99 {
				st.PeakP99 = r.LatP99
			}
			if r.LossPct > st.LossMax {
				st.LossMax = r.LossPct
			}
			if r.PausePct > st.PauseMax {
				st.PauseMax = r.PausePct
			}
			if r.ECNPct > st.ECNMax {
				st.ECNMax = r.ECNPct
			}
			if r.QDepthKB > st.QMax {
				st.QMax = r.QDepthKB
			}
		}
		out = append(out, st)
	}
	return out
}

// ReportSample 打印样本摘要：一眼看清每个端口的形态。
func ReportSample(w io.Writer, s Scenario, rows []Row) {
	fmt.Fprintf(w, "== %s · 样本摘要（%d 个端口 × %d 个采样点）==\n",
		s.Name, len(summarize(rows)), int(s.Window/s.Step))
	fmt.Fprintln(w, "端口          角色      基线p99  峰值p99   超时  loss%   pause%  ecn%   队列KB")
	for _, st := range summarize(rows) {
		peak := fmt.Sprintf("%.2f", st.PeakP99)
		if st.Timeout > 0 {
			peak = "超时"
		}
		fmt.Fprintf(w, "%-12s  %-8s  %7.2f  %8s  %3d  %5.2f  %6.1f  %5.2f  %6.0f\n",
			st.Port, st.Role, st.BaseP99, peak, st.Timeout, st.LossMax, st.PauseMax, st.ECNMax, st.QMax)
	}
	fmt.Fprintln(w)
}

// ReportVerdict 打印判读结论。格式与第 07 课「从告警到根因的四步」对齐。
func ReportVerdict(w io.Writer, v Verdict) {
	f := v.Features
	fmt.Fprintf(w, "== 判读结论：%s（置信度%s）==\n", KindTitle(v.Kind), v.Confidence)
	fmt.Fprintf(w, "特征：端口 %d 个 · 劣化 %d 个 · 超时占比 %.0f%% · loss 峰值 %.2f%% · pause 峰值 %.0f%% · ECN 峰值 %.1f%% · 时延 %.1f 倍\n",
		f.Ports, f.Affected, f.TimeoutFrac*100, f.LossMax, f.PauseMax, f.ECNMax, f.LatRatio)
	if len(f.AffectedList) > 0 {
		fmt.Fprintf(w, "劣化端口：%s\n", strings.Join(f.AffectedList, "、"))
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "依据：")
	for _, e := range v.Evidence {
		fmt.Fprintf(w, "  - %s\n", e)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "下一步：")
	for _, n := range v.Next {
		fmt.Fprintf(w, "  - %s\n", n)
	}
	fmt.Fprintln(w)
}
