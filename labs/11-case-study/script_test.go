package main

import (
	"bytes"
	"strings"
	"testing"
)

// 这些测试守的是剧本自身的一致性：
// 耗时加得上、阶段不乱序、每一步都有依据。数字一改就得重新对齐，
// 否则「40 分钟」这种结论会慢慢跟剧本脱节。

func TestBestPathTotalsForty(t *testing.T) {
	r := Execute(BestPath())
	if r.Total != 40 {
		t.Errorf("最优路径合计 %.1f 分钟，剧本与文章里写的是 40", r.Total)
	}
	if r.Units != 10 {
		t.Errorf("最优路径应有 10 步，实际 %d", r.Units)
	}
}

// 阶段累加必须等于总耗时，否则说明有步骤落到了未登记的阶段。
func TestPhaseSumsEqualTotal(t *testing.T) {
	for _, p := range []Path{BestPath(), NaivePath()} {
		r := Execute(p)
		var sum float64
		for _, pc := range r.Phases {
			sum += pc.Minutes
		}
		if sum != r.Total {
			t.Errorf("路径 %s：五段合计 %.1f，总耗时 %.1f", p.Name, sum, r.Total)
		}
	}
}

// 「先定界再缩小」这条原则要能从剧本里看出来：阶段序号不该回退。
func TestBestPathPhaseMonotonic(t *testing.T) {
	rank := map[Phase]int{}
	for i, p := range PhaseOrder {
		rank[p] = i
	}
	last := -1
	for _, s := range BestPath().Steps {
		cur := rank[s.Phase]
		if cur < last {
			t.Errorf("第 %d 步回退到了「%s」段：先定界再缩小，顺序不能返工", s.ID, s.Phase)
		}
		last = cur
	}
}

// 蛮干路径该慢。慢多少不是重点，重点是它更慢。
func TestNaiveIsSlower(t *testing.T) {
	best := Execute(BestPath()).Total
	naive := Execute(NaivePath()).Total
	if naive <= best {
		t.Fatalf("蛮干路径 %.1f 分钟不该快于最优路径 %.1f 分钟", naive, best)
	}
	if naive < best*1.5 {
		t.Errorf("差异太小（%.1f vs %.1f），对照失去说服力", naive, best)
	}
}

// 每一步都必须有判据。缺判据的步骤等于「凭感觉」，那是要被这套方法排除的做法。
func TestEveryStepHasCriterion(t *testing.T) {
	for _, s := range append(BestPath().Steps, NaivePath().Steps...) {
		if s.Question == "" || s.Seen == "" || s.Judgment == "" || s.Cost <= 0 {
			t.Errorf("第 %d 步字段不全：%+v", s.ID, s)
		}
		if s.Phase == PhaseConfirm || s.Phase == PhaseNarrow {
			if s.Criterion == "" {
				t.Errorf("第 %d 步在「%s」段却没有判据", s.ID, s.Phase)
			}
		}
	}
}

// 案例里那条关键读数不能改：lane 2 掉 6 dB 是「劣化不是抖动」的判据来源。
func TestCaseKeepsLaneEvidence(t *testing.T) {
	var lane Step
	for _, s := range BestPath().Steps {
		if strings.Contains(s.Seen, "lane") {
			lane = s
		}
	}
	if !strings.Contains(lane.Seen, "-8.9") {
		t.Errorf("案例如今缺了 lane 级光功率读数：%q", lane.Seen)
	}
	if lane.Criterion == "" {
		t.Error("lane 那一步必须写明判据（单 lane 偏离多少算劣化）")
	}
}

// 真环境复刻要诚实：不能假装十步都能跑。
func TestLivePlanIsHonest(t *testing.T) {
	plan := LivePlan()
	if len(plan) != 10 {
		t.Fatalf("真环境复刻应覆盖 10 步，实际 %d", len(plan))
	}
	can, cannot := 0, 0
	for _, l := range plan {
		if l.CanRun {
			can++
		} else {
			cannot++
		}
		if l.How == "" || l.Where == "" {
			t.Errorf("第 %d 步缺少落法说明", l.ID)
		}
	}
	if cannot == 0 {
		t.Error("DOM 与 INT 在容器里做不出来，不该声称全能复刻")
	}
	if can < 5 {
		t.Errorf("可复刻的步骤只有 %d 步，太少", can)
	}
}

// 中文表格对齐靠 dispWidth，错了整张表会歪。
func TestPadHandlesWideRunes(t *testing.T) {
	if got := dispWidth("发现"); got != 4 {
		t.Errorf("「发现」显示宽度应为 4，实际 %d", got)
	}
	if got := dispWidth("best"); got != 4 {
		t.Errorf("best 显示宽度应为 4，实际 %d", got)
	}
	if got := dispWidth(pad("发现", 8)); got != 8 {
		t.Errorf("pad 后宽度应为 8，实际 %d", got)
	}
	if got := dispWidth(pad("很长很长很长很长很长", 6)); got != 6 {
		t.Errorf("超宽截断后宽度应为 6，实际 %d", got)
	}
}

// 输出里不该出现未替换的模板痕迹或空行堆积。
func TestCompareOutputIsClean(t *testing.T) {
	var buf bytes.Buffer
	ReportCompare(&buf, Execute(BestPath()), Execute(NaivePath()))
	out := buf.String()
	if strings.Contains(out, "%!") {
		t.Errorf("格式化出错：%s", out)
	}
	if strings.Contains(out, "\n\n\n") {
		t.Error("输出里有连续空行")
	}
}
