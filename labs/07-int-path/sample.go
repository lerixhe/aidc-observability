package main

import (
	"fmt"
	"math"
	"math/rand"
)

// 采样实验的默认参数，取自课程 12 的量级估算。
const (
	DefaultPktRate = 240000 // 每秒报文数（400 Gb/s，均值 1500 字节）
	DefaultBurstUs = 200    // 一次微突发的持续时长
)

// SampleRates 是要对照的哈希采样率。
var SampleRates = []int{256, 1024, 4096}

// CaptureProb 返回在给定采样率下，一次微突发里至少被采到一个报文的概率。
//
// 哈希采样近似独立同分布，所以 p = 1 - (1 - 1/k)^m，
// m 是突发窗口内的报文数。
func CaptureProb(pktRate int, burstUs int, rate int) float64 {
	m := float64(pktRate) * float64(burstUs) / 1e6
	if rate <= 0 {
		return 0
	}
	q := 1 - 1/float64(rate)
	return 1 - math.Pow(q, m)
}

// BurstPkts 返回一次微突发里的报文数。
func BurstPkts(pktRate int, burstUs int) int {
	return int(float64(pktRate) * float64(burstUs) / 1e6)
}

// SampleRow 是一行采样对照结果。
type SampleRow struct {
	Rate      int
	BurstPkts int
	Analytic  float64 // 解析概率
	Measured  float64 // 蒙特卡洛实测
}

// RunSampling 用固定种子做蒙特卡洛，核对解析公式，并给出「连续 n 次突发全错过」的概率。
func RunSampling(pktRate, burstUs int, seed int64, trials int) []SampleRow {
	m := BurstPkts(pktRate, burstUs)
	rows := make([]SampleRow, 0, len(SampleRates))
	for _, k := range SampleRates {
		rng := rand.New(rand.NewSource(seed))
		hit := 0
		for t := 0; t < trials; t++ {
			caught := false
			for i := 0; i < m; i++ {
				if rng.Intn(k) == 0 {
					caught = true
					break
				}
			}
			if caught {
				hit++
			}
		}
		rows = append(rows, SampleRow{
			Rate:      k,
			BurstPkts: m,
			Analytic:  CaptureProb(pktRate, burstUs, k),
			Measured:  float64(hit) / float64(trials),
		})
	}
	return rows
}

// MissStreak 返回连续 n 次突发都没被采到的概率，用来回答「这套采样能不能当监控用」。
func MissStreak(prob float64, n int) float64 {
	return math.Pow(1-prob, float64(n))
}

// OverheadPct 返回 INT 元数据相对报文的带宽开销（百分比）。
func OverheadPct(payloadBytes, hops int) float64 {
	if payloadBytes <= 0 {
		return 0
	}
	return float64(WireBytes(hops)) / float64(payloadBytes) * 100
}

// SamplingSummary 生成报告文末的一句话结论。
func SamplingSummary(rows []SampleRow) string {
	if len(rows) == 0 {
		return "无采样数据"
	}
	lo, hi := rows[len(rows)-1], rows[0]
	return fmt.Sprintf("采样从 1:%d 放到 1:%d，一次微突发的捕获率从 %.1f%% 掉到 %.1f%%",
		hi.Rate, lo.Rate, hi.Analytic*100, lo.Analytic*100)
}
