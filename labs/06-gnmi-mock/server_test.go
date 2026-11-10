package main

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"
)

func TestAccumulateCycle(t *testing.T) {
	period := 20 * time.Second
	start := 10 * time.Second
	length := 1500 * time.Millisecond
	const rate = 400.0

	cases := []struct {
		t    time.Duration
		want float64
	}{
		{0, 0},
		{10 * time.Second, 0},            // 风暴刚开始，还没计数
		{11 * time.Second, 400},          // 进了 1 s
		{11500 * time.Millisecond, 600},  // 风暴结束
		{15 * time.Second, 600},          // 风暴后不变
		{20 * time.Second, 600},          // 刚好一个周期
		{20500 * time.Millisecond, 600},  // 第二周期风暴未开始
		{21500 * time.Millisecond, 600},  // 仍在第一周期的累积里
		{30 * time.Second, 600},          // 第二周期风暴刚到
		{31500 * time.Millisecond, 1200}, // 第二周期风暴结束
		{40 * time.Second, 1200},         // 两轮风暴
	}
	for _, c := range cases {
		got := accumulate(c.t, period, start, length, rate)
		if got != c.want {
			t.Errorf("accumulate(t=%v) = %v，期望 %v", c.t, got, c.want)
		}
	}
}

func TestWatermarkPlateau(t *testing.T) {
	sig := DefaultSignal()
	// 风暴中段应该在爬升
	mid := sig.watermark(sig.StormStart + sig.StormLen/2)
	if mid <= 32768 || mid >= 204800 {
		t.Errorf("风暴中段水位 %d 应处于两端之间", mid)
	}
	// 风暴末尾进入平台
	tail := sig.watermark(sig.StormStart + sig.StormLen*9/10)
	if tail != 204800 {
		t.Errorf("风暴末段水位 %d，期望贴着峰值 204800", tail)
	}
	// 风暴之外回落到稳态
	if got := sig.watermark(sig.StormStart - time.Second); got != 32768 {
		t.Errorf("风暴之外水位 %d，期望 32768", got)
	}
}

func TestMatchLeaves(t *testing.T) {
	cases := []struct {
		pattern string
		wantMin int
	}{
		{"/interfaces/interface[name=Ethernet0]/state/counters", 4},
		{"/interfaces/interface[name=Ethernet0]/state/oper-status", 1},
		{"/interfaces/interface[name=*]/state/oper-status", 2},
		{"/interfaces/.../state/counters/out-octets", 2},
		{"/qos/interfaces/interface[id=Ethernet0]/output/queues/queue[name=3]/state/watermark-bytes", 1},
	}
	for _, c := range cases {
		got, err := MatchLeaves(c.pattern)
		if err != nil {
			t.Fatalf("匹配 %q 失败：%v", c.pattern, err)
		}
		if len(got) < c.wantMin {
			t.Errorf("路径 %q 匹配到 %d 条，期望至少 %d 条", c.pattern, len(got), c.wantMin)
		}
	}
}

func TestMatchLeavesRejectsBadPath(t *testing.T) {
	if _, err := MatchLeaves(""); err == nil {
		t.Error("空路径应当报错")
	}
	if _, err := MatchLeaves("interfaces/interface"); err == nil {
		t.Error("不以 / 开头的路径应当报错")
	}
	got, err := MatchLeaves("/no/such/path")
	if err != nil {
		t.Fatalf("不该报错：%v", err)
	}
	if len(got) != 0 {
		t.Errorf("不存在的路径匹配到 %d 条", len(got))
	}
}

func TestKindsCoverAllThreeCategories(t *testing.T) {
	seen := map[Kind]bool{}
	for _, lf := range Leaves() {
		seen[lf.Kind] = true
	}
	for _, k := range []Kind{KindCounter, KindGauge, KindState} {
		if !seen[k] {
			t.Errorf("指标树里缺少 %s 类叶子", k)
		}
	}
}

func TestSubscribeSSE(t *testing.T) {
	srv := newLocalServer(NewServer(DefaultSignal(), 60).Handler())
	defer srv.Close()

	sub := Subscribe(srv.URL, pathPause, "sample", time.Second, 5*time.Second)
	if sub.Err != nil {
		t.Fatalf("订阅失败：%v", sub.Err)
	}
	if len(sub.Updates) < 5 {
		t.Errorf("收到 %d 条更新，5 s / 1 s 至少该有 5 条", len(sub.Updates))
	}
	if sub.Bytes == 0 {
		t.Error("没有统计到字节数")
	}
	// 计数器必须单调不减，否则解析把值弄错了
	for i := 1; i < len(sub.Updates); i++ {
		if sub.Updates[i].Value < sub.Updates[i-1].Value {
			t.Fatalf("计数器回退：%v -> %v", sub.Updates[i-1].Value, sub.Updates[i].Value)
		}
	}
}

func TestSubscribeOnChangeSuppressesDuplicates(t *testing.T) {
	srv := newLocalServer(NewServer(DefaultSignal(), 60).Handler())
	defer srv.Close()

	iv := time.Second
	dur := 10 * time.Second

	sample := subscribeFast(srv.URL, pathOperStatus, "sample", iv, dur)
	change := subscribeFast(srv.URL, pathOperStatus, "on_change", iv, dur)
	if sample.Err != nil || change.Err != nil {
		t.Fatalf("订阅失败：%v / %v", sample.Err, change.Err)
	}
	if len(change.Updates) >= len(sample.Updates) {
		t.Errorf("ON_CHANGE 收到 %d 条，SAMPLE 收到 %d 条，前者不该更多",
			len(change.Updates), len(sample.Updates))
	}
	// 但状态变化必须一条不漏
	if d1, d2 := countState(sample.Updates, "DOWN"), countState(change.Updates, "DOWN"); d1 != d2 {
		t.Errorf("SAMPLE 检出 %d 次 DOWN，ON_CHANGE 检出 %d 次，不该有差别", d1, d2)
	}
}

func TestGetEndpoint(t *testing.T) {
	srv := newLocalServer(NewServer(DefaultSignal(), 1).Handler())
	defer srv.Close()

	resp, err := httpGet(srv.URL + "/get?path=/interfaces/interface[name=Ethernet0]/state/oper-status&t=0")
	if err != nil {
		t.Fatalf("Get 失败：%v", err)
	}
	var body struct {
		Updates []struct {
			Path  string `json:"path"`
			Value any    `json:"value"`
		} `json:"updates"`
	}
	if err := json.Unmarshal(resp, &body); err != nil {
		t.Fatalf("响应不是合法 JSON：%v", err)
	}
	if len(body.Updates) != 1 {
		t.Fatalf("Get 返回 %d 条，期望 1 条", len(body.Updates))
	}
	if body.Updates[0].Value != "UP" {
		t.Errorf("t=0 的状态是 %v，期望 UP", body.Updates[0].Value)
	}
}

func TestGetEndpoint404(t *testing.T) {
	srv := newLocalServer(NewServer(DefaultSignal(), 1).Handler())
	defer srv.Close()
	_, code, err := httpGetStatus(srv.URL + "/get?path=/no/such/path")
	if err != nil {
		t.Fatalf("请求失败：%v", err)
	}
	if code != 404 {
		t.Errorf("不存在的路径返回 %d，期望 404", code)
	}
}

func TestBenchFidelityStory(t *testing.T) {
	b, err := RunBench(20*time.Second, 60)
	if err != nil {
		t.Fatalf("对照实验失败：%v", err)
	}

	find := func(mode string, iv time.Duration) CounterRow {
		for _, r := range b.Counter {
			if r.Mode == mode && r.Interval == iv {
				return r
			}
		}
		t.Fatalf("没找到 %s %s 的结果", mode, iv)
		return CounterRow{}
	}

	// 1 s SAMPLE 必须抓住峰值
	if r := find("SAMPLE", time.Second); r.Fidelity < 0.95 {
		t.Errorf("SAMPLE 1s 计数器保真度 %.0f%%，1.5 s 的风暴应该抓得到", r.Fidelity*100)
	}
	// 10 s SAMPLE 必然被平均掉
	if r := find("SAMPLE", 10*time.Second); r.Fidelity > 0.5 {
		t.Errorf("SAMPLE 10s 保真度 %.0f%%，长间隔不该保真", r.Fidelity*100)
	}
	// 间隔越大，保真度越低
	prev := 1.1
	for _, iv := range benchIntervals {
		r := find("SAMPLE", iv)
		if r.Fidelity > prev+1e-9 {
			t.Errorf("间隔 %v 保真度 %.2f 反而高于更小间隔的 %.2f", iv, r.Fidelity, prev)
		}
		prev = r.Fidelity
	}
	// 状态量：ON_CHANGE 必须满分，且比 SAMPLE 省得多
	var oc, sm StateRow
	for _, r := range b.State {
		if r.Interval == time.Second && r.Mode == "ON_CHANGE" {
			oc = r
		}
		if r.Interval == time.Second && r.Mode == "SAMPLE" {
			sm = r
		}
	}
	if oc.Recall < 1 {
		t.Errorf("ON_CHANGE 1s 的 DOWN 召回率 %.0f%%，状态量不该漏", oc.Recall*100)
	}
	if oc.Bytes >= sm.Bytes {
		t.Errorf("ON_CHANGE 用 %d 字节，SAMPLE 用 %d 字节，前者应当更省", oc.Bytes, sm.Bytes)
	}
}

func TestTrueDownCount(t *testing.T) {
	// 20 s 内每 7 s 抖一次：3、10、17 秒各一次
	if got := trueDownCount(DefaultSignal(), 20*time.Second); got != 3 {
		t.Errorf("20 s 内真实 DOWN 次数 %d，期望 3", got)
	}
}

func TestCompilePathPrefix(t *testing.T) {
	re, err := compilePath("/interfaces/interface[name=Ethernet0]")
	if err != nil {
		t.Fatal(err)
	}
	if !re.MatchString("/interfaces/interface[name=Ethernet0]/state/counters/out-octets") {
		t.Error("前缀路径应该能匹配到子树下的叶子")
	}
	if re.MatchString("/interfaces/interface[name=Ethernet1]/state/oper-status") {
		t.Error("不该匹配到别的接口")
	}
}

func httpGet(url string) ([]byte, error) {
	body, _, err := httpGetStatus(url)
	return body, err
}

func httpGetStatus(url string) ([]byte, int, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	return b, resp.StatusCode, err
}
