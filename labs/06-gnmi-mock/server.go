package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// Server 是一台「模拟交换机」的遥测出口。
//
// 传输层用 HTTP + SSE，不是真的 gRPC + Protobuf。这样零依赖、`go run .` 就能跑，
// 代价是真 gnmic 连不上它。语义（路径语法、订阅模式、计数器语义）按 gNMI 对齐，
// 见 README 第 3 节。
type Server struct {
	sig   *Signal
	speed float64 // 虚拟时间相对真实时间的倍率，用于把 30 秒信号在 0.5 秒内跑完
	mu    sync.Mutex
	subs  int
}

// NewServer 建一台模拟设备。speed=1 时按真实节奏推，speed=60 时 60 倍速。
func NewServer(sig *Signal, speed float64) *Server {
	if speed <= 0 {
		speed = 1
	}
	return &Server{sig: sig, speed: speed}
}

// Handler 挂上能力查询、Get、Subscribe 三个端点。
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/capabilities", s.handleCapabilities)
	mux.HandleFunc("/get", s.handleGet)
	mux.HandleFunc("/subscribe", s.handleSubscribe)
	mux.HandleFunc("/", s.handleIndex)
	return mux
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `{
  "target": "mock-leaf1",
  "endpoints": ["/capabilities", "/get?path=...", "/subscribe?path=...&mode=sample|on_change&interval=1s&duration=30s"],
  "note": "HTTP+SSE 实现，语义对齐 gNMI Subscribe，不是 gRPC"
}`)
}

func (s *Server) handleCapabilities(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{
		"vendor": "aidc-observability",
		"model":  "mock-leaf1",
		"supportedModels": []string{
			"openconfig-interfaces",
			"openconfig-qos",
		},
		"supportedEncodings": []string{"json_ietf"},
		"supportedModes":     []string{"sample", "on_change"},
		"note":               "教学用最小能力集，真实设备会多得多",
	})
}

func (s *Server) handleGet(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		path = "/interfaces/.../state/counters"
	}
	leaves, err := MatchLeaves(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(leaves) == 0 {
		http.Error(w, "路径没有匹配任何叶子："+path, http.StatusNotFound)
		return
	}
	t := parseDuration(r.URL.Query().Get("t"), 0)
	updates := make([]map[string]any, 0, len(leaves))
	for _, lf := range leaves {
		updates = append(updates, map[string]any{
			"path":  lf.Path,
			"value": lf.Value(s.sig, t),
			"kind":  lf.Kind.String(),
			"ts_ms": t.Milliseconds(),
		})
	}
	writeJSON(w, map[string]any{"updates": updates})
}

// handleSubscribe 是 SSE 版 Subscribe：
//
//	mode=sample     每 interval 推一次全部匹配叶子
//	mode=on_change  每 interval 检查一次，值变了才推
func (s *Server) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	path := q.Get("path")
	if path == "" {
		http.Error(w, "缺少 path 参数", http.StatusBadRequest)
		return
	}
	leaves, err := MatchLeaves(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(leaves) == 0 {
		http.Error(w, "路径没有匹配任何叶子："+path, http.StatusNotFound)
		return
	}

	mode := q.Get("mode")
	if mode == "" {
		mode = "sample"
	}
	if mode != "sample" && mode != "on_change" {
		http.Error(w, "mode 只支持 sample / on_change", http.StatusBadRequest)
		return
	}
	interval := parseDuration(q.Get("interval"), time.Second)
	if interval <= 0 {
		http.Error(w, "interval 必须为正", http.StatusBadRequest)
		return
	}
	duration := parseDuration(q.Get("duration"), 30*time.Second)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "服务端不支持流式响应", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Mock-Mode", mode)
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	s.mu.Lock()
	s.subs++
	s.mu.Unlock()

	last := map[string]any{}
	start := time.Now()
	// fast=1：不等真实时间，直接把整段时间点推完。
	// 通道仍是真 HTTP + SSE，只是服务端不睡 —— bench 靠它把 20 s 信号跑成毫秒级。
	fast := q.Get("fast") == "1"
	tick := time.NewTicker(time.Duration(float64(interval) / s.speed))
	if fast {
		tick.Stop()
	}
	defer tick.Stop()

	next := time.Duration(0)
	for {
		var t time.Duration
		if fast {
			t = next
			next += interval
		} else {
			t = virtualElapsed(start, s.speed)
		}
		if t > duration {
			return
		}
		for _, lf := range leaves {
			v := lf.Value(s.sig, t)
			if mode == "on_change" {
				if prev, ok := last[lf.Path]; ok && fmt.Sprint(prev) == fmt.Sprint(v) {
					continue
				}
				last[lf.Path] = v
			}
			writeSSE(w, map[string]any{
				"path":  lf.Path,
				"value": v,
				"kind":  lf.Kind.String(),
				"ts_ms": t.Milliseconds(),
			})
		}
		flusher.Flush()

		if fast {
			continue
		}
		select {
		case <-r.Context().Done():
			return
		case <-tick.C:
		}
	}
}

// virtualElapsed 把真实流逝时间换成虚拟时间。
func virtualElapsed(start time.Time, speed float64) time.Duration {
	return time.Duration(float64(time.Since(start)) * speed)
}

func writeSSE(w http.ResponseWriter, payload map[string]any) {
	b, err := json.Marshal(payload)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: update\ndata: %s\n\n", b)
}

func writeJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

func parseDuration(s string, def time.Duration) time.Duration {
	if s == "" {
		return def
	}
	if d, err := time.ParseDuration(s); err == nil {
		return d
	}
	// 允许纯数字：按秒解释
	if n, err := strconv.ParseFloat(s, 64); err == nil {
		return time.Duration(n * float64(time.Second))
	}
	return def
}
