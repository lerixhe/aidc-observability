package main

import (
	"math"
	"testing"
)

func TestECNProbThreeSegments(t *testing.T) {
	cfg := DefaultConfig()
	cfg.EgressBuf = 100
	cfg.KminRatio = 0.2
	cfg.KmaxRatio = 0.8

	cases := []struct {
		q    float64
		want float64
	}{
		{0, 0},
		{20, 0},   // 正好 Kmin，不标
		{50, 0.5}, // 区间中点，概率 50%
		{80, 1},   // 到 Kmax 全标
		{100, 1},
	}
	for _, c := range cases {
		if got := ecnProb(c.q, &cfg); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("ecnProb(q=%v) = %v，期望 %v", c.q, got, c.want)
		}
	}
}

// 有 PFC 时不丢包，没 PFC 时丢包。这是本 lab 的第一条结论。
func TestPFCKeepsLossless(t *testing.T) {
	withPFC := DefaultConfig()
	withPFC.Mode = "incast"
	withPFC.PFC = true
	withPFC.Duration = 800
	s1 := NewSim(withPFC)
	s1.Run()
	if s1.stats.DroppedKB > 0 {
		t.Errorf("启用 PFC 仍有 %.0f KB 丢弃，缓冲参数或 pause 逻辑有问题", s1.stats.DroppedKB)
	}
	if s1.stats.PauseFrames == 0 {
		t.Error("incast 场景下 PFC 一帧都没发，水位判定没生效")
	}

	noPFC := withPFC
	noPFC.PFC = false
	s2 := NewSim(noPFC)
	s2.Run()
	if s2.stats.DroppedKB <= 0 {
		t.Error("关闭 PFC 后仍然零丢弃，说明缓冲大到掩盖了过载")
	}
	if s2.stats.PauseFrames != 0 {
		t.Error("关闭 PFC 后不应有 pause 帧")
	}
}

// 队列降到 XON 以下，端口必须解冻，否则反压会变成永久死锁。
func TestXONResumesUpstream(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Mode = "incast"
	cfg.PFC = true
	cfg.Duration = 400
	cfg.PauseQuota = 500 // 配额长到不可能自动到期，只能靠 XON 解除

	s := NewSim(cfg)
	s.Run()
	if s.stats.PauseResumes == 0 {
		t.Error("从未发出 XON，水位回落后没解冻上游")
	}
	if s.stats.MaxPauseUS >= cfg.Duration {
		t.Error("端口整轮都被冻结，watchdog 或 XON 逻辑失效")
	}
}

// DCQCN 的作用就是让 PFC 少触发：ECN 是主防线，PFC 是最后防线。
func TestDCQCNReducesPauseFrames(t *testing.T) {
	base := DefaultConfig()
	base.Mode = "incast"
	base.PFC = true
	base.Duration = 1000

	without := NewSim(base)
	without.Run()

	closed := base
	closed.DCQCN = true
	loop := NewSim(closed)
	loop.Run()

	if loop.stats.PauseFrames >= without.stats.PauseFrames {
		t.Errorf("开 DCQCN 后 pause 帧 %d 不少于不开的 %d，反馈闭环没起作用",
			loop.stats.PauseFrames, without.stats.PauseFrames)
	}
}

// pause 帧传播越慢，在途流量越多，缓冲越挡不住。
func TestPauseDelayIncreasesLoss(t *testing.T) {
	fast := DefaultConfig()
	fast.Mode = "incast"
	fast.Duration = 800
	fast.PauseDelay = 1

	slow := fast
	slow.PauseDelay = 40

	a := NewSim(fast)
	a.Run()
	b := NewSim(slow)
	b.Run()

	if b.stats.DroppedKB <= a.stats.DroppedKB {
		t.Errorf("pause 延迟从 1μs 拉到 40μs，丢弃 %.0f KB 没有明显增加（基准 %.0f KB）",
			b.stats.DroppedKB, a.stats.DroppedKB)
	}
}

// 队头阻塞：无辜流拿到的吞吐应显著低于它本可以拿到的量。
func TestHolBlocksInnocentFlow(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Mode = "hol"
	cfg.PFC = true
	cfg.Duration = 600

	s := NewSim(cfg)
	s.Run()
	got := safeDiv(s.stats.F2SentKB, s.stats.F2WantedKB)
	if got > 0.5 {
		t.Errorf("无辜流 F2 达成率 %.0f%%，队头阻塞没体现出来", got*100)
	}
	if s.stats.F2WantedKB <= 0 {
		t.Error("没有记录 F2 的需求量，无法算受害者损失")
	}
}

func TestValidateRejectsBadWatermarks(t *testing.T) {
	cfg := DefaultConfig()
	cfg.XONRatio = 0.8
	cfg.XOFFRatio = 0.5
	if err := validate(cfg); err == nil {
		t.Error("xon > xoff 应当被拒绝")
	}

	cfg = DefaultConfig()
	cfg.KminRatio = 0.9
	cfg.KmaxRatio = 0.3
	if err := validate(cfg); err == nil {
		t.Error("kmin > kmax 应当被拒绝")
	}

	cfg = DefaultConfig()
	cfg.Mode = "unknown"
	if err := validate(cfg); err == nil {
		t.Error("未知 mode 应当被拒绝")
	}
}

// 同样的随机种子必须跑出同样的结果，否则 expected/ 没法当自校验基准。
func TestDeterministic(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Mode = "idc"
	cfg.Duration = 200

	a := NewSim(cfg)
	a.Run()
	b := NewSim(cfg)
	b.Run()

	if a.stats.SentKB != b.stats.SentKB || a.stats.ECNMarkedKB != b.stats.ECNMarkedKB {
		t.Errorf("同种子两次运行结果不一致：%v vs %v", a.stats.SentKB, b.stats.SentKB)
	}
}
