package main

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Send 是一条 TRACE 记录：某一步、某个 rank 向 peer 发了多少字节、耗时多久。
type Send struct {
	Step    int
	OpCount int
	Rank    int
	Peer    int
	Bytes   float64
	DurUS   float64
}

// Log 是一份解析后的 NCCL 日志。
type Log struct {
	NRanks   int
	MsgBytes float64
	Ring     []int // 来自日志的 Ring 行，可能为空
	Sends    []Send

	ParsedLines  int
	IgnoredLines int
	ANSIStripped int
}

var (
	// [0] NCCL INFO nranks 8 rank 0 comm 0x7f
	reNRanks = regexp.MustCompile(`NCCL INFO\s+nranks\s+(\d+)\s+rank\s+(\d+)`)
	// [0] NCCL INFO Ring 00 : 0 -> 2 -> 4 -> 0
	reRing = regexp.MustCompile(`NCCL INFO\s+Ring\s+\d+\s*:\s*([\d\s\->]+)`)
	// [0] NCCL INFO opCount 1 allReduce bytes 16777216
	reOp = regexp.MustCompile(`NCCL INFO\s+opCount\s+(\d+)\s+(\w+)\s+bytes\s+(\d+)`)
	// [0] NCCL TRACE step 1 opCount 1 rank 0 peer 1 bytes 14680064 dur_us 1123
	reTrace = regexp.MustCompile(`NCCL TRACE\s+step\s+(\d+)\s+opCount\s+(\d+)\s+rank\s+(\d+)\s+peer\s+(\d+)\s+bytes\s+([\d.]+)\s+dur_us\s+([\d.]+)`)
	// 终端里带颜色的日志会插 ANSI 转义码，先剥掉
	reANSI = regexp.MustCompile("\x1b\\[[0-9;]*[a-zA-Z]")
)

// Parse 读日志并还原出 Rank 间的发送关系。
func Parse(r io.Reader) (*Log, error) {
	l := &Log{}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	for sc.Scan() {
		line := sc.Text()
		if strings.ContainsRune(line, 0x1b) {
			line = reANSI.ReplaceAllString(line, "")
			l.ANSIStripped++
		}
		if strings.HasPrefix(strings.TrimSpace(line), "#") || strings.TrimSpace(line) == "" {
			continue
		}

		switch {
		case reTrace.MatchString(line):
			m := reTrace.FindStringSubmatch(line)
			s := Send{
				Step:    atoi(m[1]),
				OpCount: atoi(m[2]),
				Rank:    atoi(m[3]),
				Peer:    atoi(m[4]),
				Bytes:   atof(m[5]),
				DurUS:   atof(m[6]),
			}
			l.Sends = append(l.Sends, s)
			l.ParsedLines++

		case reRing.MatchString(line):
			m := reRing.FindStringSubmatch(line)
			l.Ring = parseRing(m[1])
			l.ParsedLines++

		case reNRanks.MatchString(line):
			m := reNRanks.FindStringSubmatch(line)
			l.NRanks = atoi(m[1])
			l.ParsedLines++

		case reOp.MatchString(line):
			m := reOp.FindStringSubmatch(line)
			// 只认第一个 opCount 的声明，用作每步消息大小的基准
			if l.MsgBytes == 0 {
				l.MsgBytes = atof(m[3])
			}
			l.ParsedLines++

		default:
			l.IgnoredLines++
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(l.Sends) == 0 {
		return nil, fmt.Errorf("日志里没有一条 TRACE 记录，检查 NCCL_DEBUG_SUBSYS 是否包含 TRACE")
	}
	if l.NRanks == 0 {
		// 日志没声明 nranks 时，从 rank 编号反推
		hi := 0
		for _, s := range l.Sends {
			if s.Rank > hi {
				hi = s.Rank
			}
			if s.Peer > hi {
				hi = s.Peer
			}
		}
		l.NRanks = hi + 1
	}
	return l, nil
}

func parseRing(s string) []int {
	fields := strings.FieldsFunc(s, func(r rune) bool {
		return r == ' ' || r == '-' || r == '>' || r == '\t'
	})
	ring := make([]int, 0, len(fields))
	for _, f := range fields {
		n, err := strconv.Atoi(strings.TrimSpace(f))
		if err != nil {
			continue
		}
		ring = append(ring, n)
	}
	// 真实日志常写成 "0 -> 1 -> 2 -> 0"，末尾重复了首元素，去掉
	if len(ring) > 1 && ring[len(ring)-1] == ring[0] {
		ring = ring[:len(ring)-1]
	}
	return ring
}

// RebuildRing 在没有 Ring 行时，从 peer 关系反推环顺序。
//
// 这条路径是本 lab 的关键演示：流量矩阵本身就是「编排的账本」，
// 即使拿不到 NCCL 自报的拓扑，也能从「谁发给谁」把它拼回来。
func (l *Log) RebuildRing() []int {
	next := map[int]int{}
	for _, s := range l.Sends {
		next[s.Rank] = s.Peer
	}
	if len(next) == 0 || len(next) != l.NRanks {
		return nil
	}
	start := 0
	for r := range next {
		if r < start {
			start = r
		}
	}
	ring := []int{start}
	cur := start
	for len(ring) < l.NRanks {
		nxt, ok := next[cur]
		if !ok {
			return nil
		}
		ring = append(ring, nxt)
		cur = nxt
		if cur == start {
			return nil // 提前闭合成小环，说明数据不完整
		}
	}
	if next[cur] != start {
		return nil
	}
	return ring
}

// Steps 返回出现过的 step，升序。
func (l *Log) Steps() []int {
	seen := map[int]bool{}
	out := []int{}
	for _, s := range l.Sends {
		if !seen[s.Step] {
			seen[s.Step] = true
			out = append(out, s.Step)
		}
	}
	sort.Ints(out)
	return out
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func atof(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}
