package main

import (
	"fmt"
	"math/rand"
	"strings"
)

// 本文件生成一份「教学用 NCCL 日志」。
//
// 行格式刻意对齐 NCCL 真实输出的风格（[rank] NCCL LEVEL ...），
// 但字段是本 lab 精简过的，能覆盖还原流量矩阵所需的最小信息集。
// 真实环境里用 NCCL_DEBUG=INFO + NCCL_DEBUG_SUBSYS=INIT,COLL,TRACE 采集，
// 再按自己版本的输出调整 parse.go 的正则。

// GenParams 控制生成哪一份日志。
type GenParams struct {
	Ranks      int
	MsgBytes   float64 // 每步 AllReduce 的消息大小
	Steps      int
	PortGBps   float64 // 每 rank 单端口线速
	Ring       []int   // 环顺序，空则用默认的拓扑感知顺序
	FaultStep  [2]int  // 故障步区间（含两端），[0,0] 表示没有故障
	FaultRank  int
	FaultScale float64 // 故障 rank 的耗时倍数
	Jitter     float64 // 正常抖动的标准差，0.05 = ±5%
	Seed       int64
}

// DefaultGenParams 给出一份 8 rank、带一次 straggler 的日志。
func DefaultGenParams() GenParams {
	return GenParams{
		Ranks:      8,
		MsgBytes:   16 << 20, // 16 MiB，7B 模型的梯度切片量级
		Steps:      10,
		PortGBps:   25, // 200 Gbps
		FaultStep:  [2]int{7, 9},
		FaultRank:  3,
		FaultScale: 2.5,
		Jitter:     0.05,
		Seed:       42,
	}
}

// defaultRing 给一个拓扑感知式的环：先跨 rail，再回填。
// 排列不连续，方便读者体会「ring 顺序决定哪条链路被打满」。
func defaultRing(n int) []int {
	ring := make([]int, 0, n)
	for i := 0; i < n; i += 2 {
		ring = append(ring, i)
	}
	for i := 1; i < n; i += 2 {
		ring = append(ring, i)
	}
	return ring
}

// ringTrafficPerPeer 返回 Ring AllReduce 中每个 rank 发给下一跳的字节数。
//
// 环上跑两圈（ReduceScatter + AllGather），每圈 N-1 步，每步搬 S/N，
// 所以每个 rank 对下一跳累计发送 2(N-1)/N × S。
func ringTrafficPerPeer(msgBytes float64, n int) float64 {
	return 2 * float64(n-1) / float64(n) * msgBytes
}

// GenerateLog 渲染日志文本。
func GenerateLog(p GenParams) string {
	if len(p.Ring) == 0 {
		p.Ring = defaultRing(p.Ranks)
	}
	rng := rand.New(rand.NewSource(p.Seed))
	perPeer := ringTrafficPerPeer(p.MsgBytes, p.Ranks)
	next := make([]int, p.Ranks)
	for i, r := range p.Ring {
		next[r] = p.Ring[(i+1)%len(p.Ring)]
	}

	b := &strings.Builder{}
	fmt.Fprintf(b, "# 教学用 NCCL 日志，行格式见 labs/04-nccl-trace/README.md 第 3 节\n")
	fmt.Fprintf(b, "# 真实采集：NCCL_DEBUG=INFO NCCL_DEBUG_SUBSYS=INIT,COLL,TRACE\n")
	fmt.Fprintf(b, "[0] NCCL INFO NCCL version 2.21.5+cuda12.4\n")
	fmt.Fprintf(b, "[0] NCCL INFO nranks %d rank 0 comm 0x7f0000000000\n", p.Ranks)
	fmt.Fprintf(b, "[0] NCCL INFO Ring 00 : %s -> %d\n", joinRing(p.Ring), p.Ring[0])
	fmt.Fprintf(b, "[0] NCCL INFO opCount 1 allReduce bytes %.0f\n", p.MsgBytes)

	for step := 1; step <= p.Steps; step++ {
		for _, rank := range p.Ring {
			dur := idealDurUS(perPeer, p.PortGBps) * (1 + p.Jitter*rng.NormFloat64())
			if p.FaultRank >= 0 && rank == p.FaultRank &&
				step >= p.FaultStep[0] && step <= p.FaultStep[1] {
				dur *= p.FaultScale
			}
			fmt.Fprintf(b, "[%d] NCCL TRACE step %d opCount %d rank %d peer %d bytes %.0f dur_us %.0f\n",
				rank, step, step, rank, next[rank], perPeer, dur)
		}
	}
	return b.String()
}

// idealDurUS 把字节数换算成 μs：bytes ÷ (GB/s × 1e9) 得秒，乘 1e6 得 μs。
func idealDurUS(bytes, gbps float64) float64 {
	return bytes / (gbps * 1e9) * 1e6
}

func joinRing(ring []int) string {
	parts := make([]string, len(ring))
	for i, r := range ring {
		parts[i] = fmt.Sprintf("%d", r)
	}
	return strings.Join(parts, " -> ")
}
