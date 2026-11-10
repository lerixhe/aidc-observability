package main

import (
	"strings"
	"testing"
)

// 读仓库里的样本日志；找不到就现场生成一份，保证测试独立可跑。
func loadSample(t *testing.T) *Log {
	t.Helper()
	text := GenerateLog(DefaultGenParams())
	l, err := Parse(strings.NewReader(text))
	if err != nil {
		t.Fatalf("解析生成日志失败：%v", err)
	}
	return l
}

func TestRingTrafficPerPeer(t *testing.T) {
	const n = 8
	const s = 16 << 20
	got := ringTrafficPerPeer(float64(s), n)
	// 2 × (8-1)/8 × 16 MiB = 28 MiB
	if want := 2 * 7.0 / 8.0 * s; got != want {
		t.Errorf("每跳累计发送 %.0f，期望 %.0f", got, want)
	}
}

func TestParseBasics(t *testing.T) {
	l := loadSample(t)
	if l.NRanks != 8 {
		t.Errorf("nranks = %d，期望 8", l.NRanks)
	}
	if l.MsgBytes != 16<<20 {
		t.Errorf("消息大小 %.0f，期望 %d", l.MsgBytes, 16<<20)
	}
	if len(l.Sends) != 8*10 {
		t.Errorf("TRACE 记录 %d 条，期望 80", len(l.Sends))
	}
	if len(l.Ring) != 8 {
		t.Errorf("环长度 %d，期望 8", len(l.Ring))
	}
	if l.IgnoredLines != 1 {
		t.Errorf("忽略行数 %d，期望 1（版本号那行）", l.IgnoredLines)
	}
}

func TestParseToleratesANSIAndNoise(t *testing.T) {
	text := "\x1b[32m[0] NCCL INFO nranks 2 rank 0 comm 0x1\x1b[0m\n" +
		"garbage line\n" +
		"[0] NCCL TRACE step 1 opCount 1 rank 0 peer 1 bytes 1024 dur_us 100\n" +
		"[1] NCCL TRACE step 1 opCount 1 rank 1 peer 0 bytes 1024 dur_us 100\n"
	l, err := Parse(strings.NewReader(text))
	if err != nil {
		t.Fatalf("解析失败：%v", err)
	}
	if l.ANSIStripped != 1 {
		t.Errorf("ANSI 行计数 %d，期望 1", l.ANSIStripped)
	}
	if l.IgnoredLines != 1 {
		t.Errorf("噪音行计数 %d，期望 1", l.IgnoredLines)
	}
	if len(l.Sends) != 2 {
		t.Errorf("TRACE 记录 %d 条，期望 2", len(l.Sends))
	}
}

func TestParseRejectsEmptyLog(t *testing.T) {
	_, err := Parse(strings.NewReader("[0] NCCL INFO NCCL version 2.21.5\n"))
	if err == nil {
		t.Error("没有 TRACE 记录时应报错，提示检查 NCCL_DEBUG_SUBSYS")
	}
}

// 拿不到 Ring 行时，从「谁发给谁」反推环顺序。
func TestRebuildRingFromTraffic(t *testing.T) {
	l := loadSample(t)
	want := []int{0, 2, 4, 6, 1, 3, 5, 7}
	l.Ring = nil // 模拟日志里没有 Ring 行
	got := l.RebuildRing()
	if len(got) != len(want) {
		t.Fatalf("反推得到 %v，期望 %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("反推得到 %v，期望 %v", got, want)
		}
	}
}

func TestAnalyzeFindsStraggler(t *testing.T) {
	l := loadSample(t)
	a := Analyze(l)

	if len(a.Steps) != 10 {
		t.Fatalf("核算出 %d 步，期望 10", len(a.Steps))
	}
	if len(a.AnomalySteps) != 3 {
		t.Errorf("异常步 %v，期望 3 步（7,8,9）", a.AnomalySteps)
	}
	if a.SuspectRank != 3 {
		t.Errorf("最慢 rank 归因到 %d，期望 3", a.SuspectRank)
	}
	if !a.RingFromLog {
		t.Error("日志自带 Ring 行时 RingFromLog 应为 true")
	}
	if a.WastedRankUS() <= 0 {
		t.Error("有异常步却没算出浪费的卡时")
	}
}

// busbw = algbw × 2(N-1)/N，这是 Ring AllReduce 的核心换算。
func TestBusBWFormula(t *testing.T) {
	l := loadSample(t)
	a := Analyze(l)
	n := float64(a.N)
	for _, s := range a.Steps {
		want := s.AlgBW * 2 * (n - 1) / n
		if diff := s.BusBW - want; diff > 1e-9 || diff < -1e-9 {
			t.Errorf("step %d busbw %.3f，公式值 %.3f", s.Step, s.BusBW, want)
		}
	}
}

// 无故障日志不应报异常。
func TestCleanLogHasNoAnomaly(t *testing.T) {
	p := DefaultGenParams()
	p.FaultRank = -1
	l, err := Parse(strings.NewReader(GenerateLog(p)))
	if err != nil {
		t.Fatalf("解析失败：%v", err)
	}
	a := Analyze(l)
	if len(a.AnomalySteps) != 0 {
		t.Errorf("无故障日志报出异常步 %v", a.AnomalySteps)
	}
}

func TestParseFaultFlag(t *testing.T) {
	fs, rank, scale, err := parseFault("7-9,3,2.5")
	if err != nil {
		t.Fatalf("解析失败：%v", err)
	}
	if fs != [2]int{7, 9} || rank != 3 || scale != 2.5 {
		t.Errorf("得到 %v %d %v", fs, rank, scale)
	}
	if _, _, _, err := parseFault("7-9,3"); err == nil {
		t.Error("段数不足应当报错")
	}
	if _, _, _, err := parseFault("9-7,3,2"); err == nil {
		t.Error("to < from 应当报错")
	}
}
