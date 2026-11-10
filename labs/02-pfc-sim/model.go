package main

import (
	"math"
	"math/rand"
)

// 单位约定（整个 lab 统一，改参数时先看这里）
//
//	时间：微秒 μs
//	数据量：KB（十进制，1 KB = 1000 B）
//	速率：KB/μs，1 KB/μs = 1 GB/s = 8 Gbps
//
// 例：200 Gbps 端口 = 25 GB/s = 25 KB/μs。
//
// 模型是「流体近似」：不逐包模拟，按 KB 连续计数。
// 教学级模型，非硬件行为等价，详见 README。

const dtUS = 1.0

// Config 是模型的可调参数集合。
type Config struct {
	Mode        string  // incast / idc / hol
	PFC         bool    // 是否启用 PFC 反压
	DCQCN       bool    // 是否启用 ECN 反馈闭环（简化版 DCQCN）
	Ports       int     // 上游端口数，每个端口一个 ingress 缓冲
	PortRate    float64 // 端口线速，KB/μs
	IngressBuf  float64 // 每个 ingress 缓冲上限，KB
	EgressBuf   float64 // 出端口共享缓冲上限，KB
	SrcBuf      float64 // 每个源端网卡的发送积压上限，KB
	XOFFRatio   float64 // XOFF 高水线，占 IngressBuf 比例
	XONRatio    float64 // XON 低水线，占 IngressBuf 比例
	KminRatio   float64 // ECN 标记下限，占 EgressBuf 比例
	KmaxRatio   float64 // ECN 标记上限，占 EgressBuf 比例
	PauseQuota  float64 // 单个 pause 帧的时间配额，μs
	PauseDelay  float64 // pause 帧传到上游的延迟，μs
	CNPDelay    float64 // CE 标记回到发送端的延迟，μs
	Duration    float64 // 仿真时长，μs
	SlowFactor  float64 // hol 模式下慢目的地的可用带宽比例
	Load        float64 // idc 模式的目标平均负载，占出端口线速比例
	BurstRatio  float64 // incast 模式下同步突发占周期的比例
	Period      float64 // incast 模式下的同步周期，μs
	WatchdogUS  float64 // PFC watchdog 阈值，0 表示不启用
	IgnorePause bool    // 模拟上游忽略 pause 帧（PFC 配置不匹配），演示 watchdog 介入
	SampleUS    float64 // 采样间隔，μs
	Seed        int64
}

// DefaultConfig 给出一组「一个出端口、四个上游」的最小可观测场景。
func DefaultConfig() Config {
	return Config{
		Mode:       "incast",
		PFC:        true,
		DCQCN:      false,
		Ports:      4,
		PortRate:   25,
		IngressBuf: 256,
		EgressBuf:  192,
		SrcBuf:     4000,
		XOFFRatio:  0.5,
		XONRatio:   0.25,
		KminRatio:  0.15,
		KmaxRatio:  0.6,
		PauseQuota: 10,
		PauseDelay: 2,
		CNPDelay:   10,
		Duration:   2000,
		SlowFactor: 0.2,
		Load:       0.8,
		BurstRatio: 0.2,
		Period:     200,
		WatchdogUS: 400,
		SampleUS:   20,
		Seed:       42,
	}
}

// Sample 是一个采样点的观测快照，也是 CSV 的一行。
type Sample struct {
	T            float64 // 时刻，μs
	IngressMaxKB float64 // 各 ingress 缓冲的最大占用
	EgressKB     float64 // 出端口共享缓冲占用
	PausedPorts  int     // 处于暂停状态的端口数
	ECNMarkedKB  float64 // 本采样窗口内被 CE 标记的数据量
	DroppedKB    float64 // 本采样窗口内溢出的数据量
	SentKB       float64 // 本采样窗口内送出的数据量
}

// Stats 是整轮仿真的汇总计数。
type Stats struct {
	InjectedKB   float64
	SentKB       float64
	DroppedKB    float64
	ECNMarkedKB  float64
	PauseFrames  int
	PauseResumes int
	PausedPortUS float64 // 端口·μs 累积，用于算反压覆盖率
	MaxPauseUS   float64
	WatchdogHits int
	IngressPeak  float64
	SrcPeak      float64
	IngressSum   float64
	EgressPeak   float64
	EgressSum    float64
	F1InjectedKB float64
	F1SentKB     float64
	F2InjectedKB float64
	F2SentKB     float64
	F1WantedKB   float64 // 不受反压影响的需求量，用来算无辜流的真实损失
	F2WantedKB   float64
	Samples      int
}

// PauseCoverage 返回「有端口处于暂停状态」的时间占比（0-1）。
func (st *Stats) PauseCoverage(ports int, duration float64) float64 {
	if ports <= 0 || duration <= 0 {
		return 0
	}
	return st.PausedPortUS / (float64(ports) * duration)
}

type pauseEvent struct {
	at    float64
	port  int
	on    bool
	quota float64
}

type cnpEvent struct {
	at   float64
	port int
}

// Sim 是一台交换机加 N 条上游链路的模型。
//
//	上游端口 i ──> ingress[i] 缓冲 ──> 出端口共享缓冲 ──> 下游
//	                    ↑ pause 帧反压
type Sim struct {
	cfg Config
	rng *rand.Rand

	t          float64
	ingress    []float64
	paused     []bool
	pauseSince []float64 // 本轮暂停开始的时刻
	pauseUntil []float64 // pause 配额到期时刻
	sentPause  []bool    // 已发出 pause 帧、等待生效
	srcRate    []float64 // 每源当前发送速率（DCQCN 降速用）
	srcBacklog []float64 // 每源网卡里待发但被反压挡住的数据，KB

	egressF1 float64 // 出端口缓冲中属于「拥塞目的地」的字节
	egressF2 float64 // 出端口缓冲中属于「健康目的地」的字节

	events []pauseEvent
	cnps   []cnpEvent

	lastCNP    float64
	nextSample float64
	winSent    float64
	winMarked  float64
	winDropped float64
	samples    []Sample
	stats      Stats
}

// NewSim 按配置建好模型。
func NewSim(cfg Config) *Sim {
	s := &Sim{
		cfg:        cfg,
		rng:        rand.New(rand.NewSource(cfg.Seed)),
		ingress:    make([]float64, cfg.Ports),
		paused:     make([]bool, cfg.Ports),
		pauseSince: make([]float64, cfg.Ports),
		pauseUntil: make([]float64, cfg.Ports),
		sentPause:  make([]bool, cfg.Ports),
		srcRate:    make([]float64, cfg.Ports),
		srcBacklog: make([]float64, cfg.Ports),
	}
	for i := range s.srcRate {
		s.srcRate[i] = cfg.PortRate
	}
	return s
}

// Run 跑完整轮仿真并返回采样序列。
func (s *Sim) Run() []Sample {
	for s.t < s.cfg.Duration-1e-9 {
		s.Step()
		s.t += dtUS
	}
	return s.samples
}

// Step 推进一个时间步。顺序刻意与实际设备一致：
// 先发送腾空间，再搬运，再接收上游到达，最后看水线决定是否反压。
func (s *Sim) Step() {
	cfg := &s.cfg

	// 0) pause 配额到期自动解除（802.1Qbb 的 pause 帧带时间配额）
	for i := range s.paused {
		if s.paused[i] && s.t >= s.pauseUntil[i]-1e-9 {
			s.endPause(i)
		}
	}

	// 1) 传播中的 pause / XON 帧生效
	if len(s.events) > 0 {
		kept := s.events[:0]
		for _, e := range s.events {
			if e.at <= s.t+1e-9 {
				s.applyPause(e)
			} else {
				kept = append(kept, e)
			}
		}
		s.events = kept
	}

	// 2) 到期的 CNP 让发送端降速
	if len(s.cnps) > 0 {
		kept := s.cnps[:0]
		for _, c := range s.cnps {
			if c.at <= s.t+1e-9 {
				s.srcRate[c.port] = math.Max(s.srcRate[c.port]*(1-cfg.reduceStep()), 0.1*cfg.PortRate)
			} else {
				kept = append(kept, c)
			}
		}
		s.cnps = kept
	}

	// 3) 没收到 CNP 时定时器驱动逐步恢复（DCQCN 的 additive increase）
	if cfg.DCQCN {
		for i := range s.srcRate {
			s.srcRate[i] = math.Min(cfg.PortRate, s.srcRate[i]+cfg.PortRate*dtUS/cfg.recoverUS())
		}
	}

	// 4) 出端口按线速发送，腾出缓冲空间
	p := ecnProb(s.egressTotal(), cfg)
	f1Cap := cfg.PortRate * dtUS
	if cfg.Mode == "hol" {
		f1Cap *= cfg.SlowFactor // 慢目的地只吃得下这么多
	}
	f1Out := math.Min(s.egressF1, f1Cap)
	f2Out := math.Min(s.egressF2, math.Max(0, cfg.PortRate*dtUS-f1Out))
	s.egressF1 -= f1Out
	s.egressF2 -= f2Out
	sent := f1Out + f2Out
	s.stats.F1SentKB += f1Out
	s.stats.F2SentKB += f2Out
	s.trackSent(sent)
	if p > 0 && sent > 0 {
		marked := sent * p
		s.stats.ECNMarkedKB += marked
		s.winMarked += marked
		if cfg.DCQCN {
			s.scheduleCNP()
		}
	}

	// 5) ingress → 出端口缓冲搬运，受共享缓冲剩余空间限制
	if space := cfg.EgressBuf - s.egressTotal(); space > 0 {
		total := 0.0
		for _, v := range s.ingress {
			total += v
		}
		if total > 0 {
			fabric := float64(cfg.Ports) * cfg.PortRate * dtUS // 内部带宽充裕，不是瓶颈
			move := math.Min(space, math.Min(fabric, total))
			share := f1Share(cfg)
			for i := range s.ingress {
				take := move * s.ingress[i] / total
				s.ingress[i] -= take
				s.egressF1 += take * share
				s.egressF2 += take * (1 - share)
			}
		}
	}

	// 6) 上游按各自的流量模型注入
	s.inject()

	// 7) 溢出丢弃：缓冲满的那一刻，无损网络也会丢
	for i := range s.ingress {
		if s.ingress[i] > cfg.IngressBuf {
			over := s.ingress[i] - cfg.IngressBuf
			s.ingress[i] = cfg.IngressBuf
			s.stats.DroppedKB += over
			s.winDropped += over
		}
	}

	// 8) 看水线决定是否向上游发 pause 帧
	s.checkPause()

	// 9) PFC watchdog：暂停太久就强制恢复，用「允许丢一点」换「不永久死锁」
	s.checkWatchdog()

	// 10) 统计与采样
	s.collect()
}

// ---- 流量模型 ----

func (s *Sim) inject() {
	cfg := &s.cfg
	switch cfg.Mode {
	case "incast":
		// AllReduce 的节奏：每 period 里只有一小段在同步发送，其余在算梯度。
		// 被反压挡住的数据堆在源端网卡队列里，解冻后立刻补发 —— 这是「堆积」的新观测面。
		inBurst := math.Mod(s.t, cfg.Period) < cfg.Period*cfg.BurstRatio
		for i := 0; i < cfg.Ports; i++ {
			if inBurst {
				s.stats.F1WantedKB += cfg.PortRate * dtUS
				s.srcBacklog[i] = math.Min(cfg.SrcBuf, s.srcBacklog[i]+cfg.PortRate*dtUS)
			}
			if s.paused[i] && !cfg.IgnorePause {
				continue // 反压生效：上游冻结该优先级队列
			}
			kb := math.Min(s.sourceRateKB(i), s.srcBacklog[i])
			s.srcBacklog[i] -= kb
			s.addIngress(i, kb, 1.0)
		}

	case "hol":
		// 一半流量去慢目的地（F1），一半去健康目的地（F2），共享同一个无损队列
		for i := 0; i < cfg.Ports; i++ {
			want := cfg.PortRate * dtUS * 0.5
			s.stats.F1WantedKB += want
			s.stats.F2WantedKB += want
			s.srcBacklog[i] = math.Min(cfg.SrcBuf, s.srcBacklog[i]+2*want)
			if s.paused[i] && !cfg.IgnorePause {
				continue
			}
			kb := math.Min(s.sourceRateKB(i), s.srcBacklog[i])
			s.srcBacklog[i] -= kb
			s.addIngress(i, kb*0.5, 1.0)
			s.addIngress(i, kb*0.5, 0.0)
		}

	case "idc":
		// 传统 IDC：大量小流随机到达，平均负载可控，单流突发被排队统计吸收
		mean := cfg.Load * cfg.PortRate * dtUS / float64(cfg.Ports)
		for i := 0; i < cfg.Ports; i++ {
			if s.paused[i] && !cfg.IgnorePause {
				continue
			}
			kb := s.rng.ExpFloat64() * mean
			s.addIngress(i, kb, 1.0)
		}
	}
}

func (s *Sim) sourceRateKB(i int) float64 {
	if s.cfg.DCQCN {
		return s.srcRate[i] * dtUS
	}
	return s.cfg.PortRate * dtUS
}

func (s *Sim) addIngress(i int, kb, f1Frac float64) {
	if kb <= 0 {
		return
	}
	s.stats.InjectedKB += kb
	s.stats.F1InjectedKB += kb * f1Frac
	s.stats.F2InjectedKB += kb * (1 - f1Frac)
	s.ingress[i] += kb
}

// f1Share 返回注入流量中「去拥塞目的地」的比例。
func f1Share(cfg *Config) float64 {
	if cfg.Mode == "hol" {
		return 0.5
	}
	return 1.0
}

// ---- PFC 与 ECN ----

func (s *Sim) checkPause() {
	cfg := &s.cfg
	if !cfg.PFC {
		return
	}
	hi := cfg.IngressBuf * cfg.XOFFRatio
	lo := cfg.IngressBuf * cfg.XONRatio
	for i := range s.ingress {
		switch {
		case !s.paused[i] && !s.sentPause[i] && s.ingress[i] >= hi:
			// 只有一条上游，直接发给它；真实网络里是发给直连的上一跳
			s.events = append(s.events, pauseEvent{at: s.t + cfg.PauseDelay, port: i, on: true, quota: cfg.PauseQuota})
			s.sentPause[i] = true
			s.stats.PauseFrames++
		case s.paused[i] && s.ingress[i] <= lo:
			s.events = append(s.events, pauseEvent{at: s.t + cfg.PauseDelay, port: i, on: false})
			s.stats.PauseResumes++
		case s.paused[i] && s.pauseUntil[i]-s.t < 2*cfg.PauseDelay:
			// 配额快到期而水位没降下来，续发一帧。真实设备同样会重发，
			// 否则每到期一次就给上游开一个窗口，在途流量正好把缓冲顶爆。
			s.events = append(s.events, pauseEvent{at: s.t + cfg.PauseDelay, port: i, on: true, quota: cfg.PauseQuota})
			s.stats.PauseFrames++
		}
	}
}

func (s *Sim) applyPause(e pauseEvent) {
	if e.on {
		if !s.paused[e.port] {
			s.paused[e.port] = true
			s.pauseSince[e.port] = s.t
		}
		s.pauseUntil[e.port] = s.t + e.quota
		s.sentPause[e.port] = false
		return
	}
	s.endPause(e.port)
}

func (s *Sim) endPause(i int) {
	if !s.paused[i] {
		s.sentPause[i] = false
		return
	}
	if d := s.t - s.pauseSince[i]; d > s.stats.MaxPauseUS {
		s.stats.MaxPauseUS = d
	}
	s.paused[i] = false
	s.sentPause[i] = false
	s.pauseUntil[i] = 0
}

func (s *Sim) checkWatchdog() {
	cfg := &s.cfg
	if cfg.WatchdogUS <= 0 || !cfg.PFC {
		return
	}
	for i := range s.paused {
		if !s.paused[i] {
			continue
		}
		if s.t-s.pauseSince[i] <= cfg.WatchdogUS {
			continue
		}
		// 强制解除暂停，并丢弃该队列积压：可用性优先于完整性
		s.stats.WatchdogHits++
		s.stats.DroppedKB += s.ingress[i]
		s.winDropped += s.ingress[i]
		s.ingress[i] = 0
		s.endPause(i)
	}
}

func (s *Sim) scheduleCNP() {
	cfg := &s.cfg
	if s.t-s.lastCNP < 5 {
		return // 控制 CNP 频率，别把源端打成龟速
	}
	s.lastCNP = s.t
	for i := 0; i < cfg.Ports; i++ {
		if s.paused[i] {
			continue
		}
		s.cnps = append(s.cnps, cnpEvent{at: s.t + cfg.CNPDelay, port: i})
	}
}

// ecnProb 是「三段水线」的标记概率：低于 Kmin 不标，Kmin~Kmax 线性升，超过 Kmax 全标。
func ecnProb(q float64, cfg *Config) float64 {
	kmin := cfg.EgressBuf * cfg.KminRatio
	kmax := cfg.EgressBuf * cfg.KmaxRatio
	switch {
	case q <= kmin:
		return 0
	case q >= kmax:
		return 1
	default:
		return (q - kmin) / (kmax - kmin)
	}
}

func (c *Config) reduceStep() float64 { return 0.15 } // 收到 CNP 后降速比例
func (c *Config) recoverUS() float64  { return 400 }  // 无 CNP 时的恢复时间常数

// ---- 记账 ----

func (s *Sim) egressTotal() float64 { return s.egressF1 + s.egressF2 }

func (s *Sim) trackSent(kb float64) {
	s.stats.SentKB += kb
	s.winSent += kb
}

func (s *Sim) collect() {
	for i := range s.paused {
		if s.paused[i] {
			s.stats.PausedPortUS += dtUS
		}
	}
	for _, v := range s.srcBacklog {
		if v > s.stats.SrcPeak {
			s.stats.SrcPeak = v
		}
	}
	imax := 0.0
	for _, v := range s.ingress {
		if v > imax {
			imax = v
		}
	}
	if imax > s.stats.IngressPeak {
		s.stats.IngressPeak = imax
	}
	s.stats.IngressSum += imax
	q := s.egressTotal()
	if q > s.stats.EgressPeak {
		s.stats.EgressPeak = q
	}
	s.stats.EgressSum += q

	if s.t >= s.nextSample {
		pausedPorts := 0
		for _, p := range s.paused {
			if p {
				pausedPorts++
			}
		}
		s.samples = append(s.samples, Sample{
			T:            s.t,
			IngressMaxKB: imax,
			EgressKB:     q,
			PausedPorts:  pausedPorts,
			ECNMarkedKB:  s.winMarked,
			DroppedKB:    s.winDropped,
			SentKB:       s.winSent,
		})
		s.stats.Samples++
		s.winMarked, s.winDropped, s.winSent = 0, 0, 0
		s.nextSample += s.cfg.SampleUS
	}
}
