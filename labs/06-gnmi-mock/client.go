package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Update 是一条推送样本，字段对齐 gNMI SubscribeResponse 的语义。
type Update struct {
	Path  string  `json:"path"`
	Value float64 `json:"-"` // 数值型
	Str   string  `json:"-"` // 状态型
	Kind  string  `json:"kind"`
	TsMS  int64   `json:"ts_ms"`
}

// Subscription 是一次订阅的完整结果。
type Subscription struct {
	Updates []Update
	Bytes   int
	Err     error
}

// Subscribe 连上 mock 设备并读完一次订阅。
func Subscribe(baseURL, path, mode string, interval, duration time.Duration) *Subscription {
	return subscribeWith(baseURL, path, mode, interval, duration, false)
}

// subscribeFast 用服务端的 fast replay，不等真实时间，供对照实验使用。
func subscribeFast(baseURL, path, mode string, interval, duration time.Duration) *Subscription {
	return subscribeWith(baseURL, path, mode, interval, duration, true)
}

func subscribeWith(baseURL, path, mode string, interval, duration time.Duration, fast bool) *Subscription {
	url := fmt.Sprintf("%s/subscribe?path=%s&mode=%s&interval=%s&duration=%s",
		baseURL, path, mode, interval, duration)
	if fast {
		url += "&fast=1"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return &Subscription{Err: err}
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return &Subscription{Err: err}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return &Subscription{Err: fmt.Errorf("订阅失败 %d：%s", resp.StatusCode, strings.TrimSpace(string(body)))}
	}

	sub := &Subscription{}
	counting := &countingReader{r: resp.Body}
	sc := bufio.NewScanner(counting)
	sc.Buffer(make([]byte, 0, 16*1024), 1<<20)

	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		var raw struct {
			Path  string          `json:"path"`
			Value json.RawMessage `json:"value"`
			Kind  string          `json:"kind"`
			TsMS  int64           `json:"ts_ms"`
		}
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &raw); err != nil {
			continue
		}
		u := Update{Path: raw.Path, Kind: raw.Kind, TsMS: raw.TsMS}
		if v, ok := parseNum(raw.Value); ok {
			u.Value = v
		} else {
			u.Str = strings.Trim(string(raw.Value), `"`)
		}
		sub.Updates = append(sub.Updates, u)
	}
	sub.Err = sc.Err()
	sub.Bytes = counting.n
	return sub
}

func parseNum(raw json.RawMessage) (float64, bool) {
	var f float64
	if err := json.Unmarshal(raw, &f); err == nil {
		return f, true
	}
	return 0, false
}

type countingReader struct {
	r io.Reader
	n int
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += n
	return n, err
}

// series 抽出一条路径上的样本序列。
func (s *Subscription) series(path string) []Update {
	var out []Update
	for _, u := range s.Updates {
		if u.Path == path {
			out = append(out, u)
		}
	}
	return out
}

// peakRate 由累积计数的相邻样本反推速率峰值。
//
// 这是设备遥测的通用口径：计数器本身没意义，两次采样相减除以间隔才是速率。
// 间隔越大，速率越是被平均 —— 这正是「分钟轮询只看到平均值」的机制。
func peakRate(samples []Update) float64 {
	peak := 0.0
	for i := 1; i < len(samples); i++ {
		dt := time.Duration(samples[i].TsMS-samples[i-1].TsMS) * time.Millisecond
		r := rateBetween(uint64(samples[i-1].Value), uint64(samples[i].Value), dt)
		if r > peak {
			peak = r
		}
	}
	return peak
}

// peakValue 取 gauge 序列的峰值。
func peakValue(samples []Update) float64 {
	peak := 0.0
	for _, s := range samples {
		if s.Value > peak {
			peak = s.Value
		}
	}
	return peak
}

// countState 统计状态序列里出现过多少次目标值（相邻去重）。
func countState(samples []Update, target string) int {
	n := 0
	prev := ""
	for _, s := range samples {
		if s.Str == target && prev != target {
			n++
		}
		prev = s.Str
	}
	return n
}
