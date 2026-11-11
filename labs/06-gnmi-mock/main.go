package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"
)

func main() {
	serve := flag.Bool("serve", false, "只起 mock 设备，等外部客户端来订阅")
	addr := flag.String("addr", ":9339", "-serve 模式下的监听地址")
	speed := flag.Float64("speed", 60, "虚拟时间加速倍率，60 表示 30 s 信号 0.5 s 跑完")
	duration := flag.Duration("duration", 20*time.Second, "订阅的虚拟时长（默认一个完整风暴周期）")
	getPath := flag.String("get", "", "对本地 mock 设备做一次 Get，打印快照")
	listPaths := flag.Bool("paths", false, "列出模拟设备支持的全部叶子路径")
	oneShot := flag.String("subscribe", "", "对本地 mock 设备订阅指定路径（配合 -mode/-interval）")
	mode := flag.String("mode", "sample", "订阅模式：sample / on_change")
	interval := flag.Duration("interval", time.Second, "采样间隔")
	bench := flag.Bool("bench", true, "跑订阅策略保真度对照（默认行为）")
	flag.Parse()

	sig := DefaultSignal()

	switch {
	case *listPaths:
		for _, lf := range Leaves() {
			fmt.Printf("%-12s %s\n", lf.Kind, lf.Path)
		}
		return

	case *serve:
		fmt.Printf("mock 设备监听 %s · 加速 %.0f×\n", *addr, *speed)
		fmt.Println("试一下：")
		fmt.Printf("  curl 'http://localhost%s/capabilities'\n", *addr)
		fmt.Printf("  curl 'http://localhost%s/metrics'\n", *addr)
		fmt.Printf("  curl 'http://localhost%s/get?path=%s'\n", *addr, pathPause)
		fmt.Printf("  curl -N 'http://localhost%s/subscribe?path=%s&mode=sample&interval=1s&duration=10s'\n", *addr, pathPause)
		if err := http.ListenAndServe(*addr, NewServer(sig, *speed).Handler()); err != nil {
			fmt.Fprintf(os.Stderr, "监听失败：%v\n", err)
			os.Exit(1)
		}
		return
	}

	if *getPath != "" || *oneShot != "" {
		// 单次交互模式：内部起一个临时服务，跑完即退
		runSingle(sig, *speed, *getPath, *oneShot, *mode, *interval, *duration)
		return
	}

	if !*bench {
		flag.Usage()
		return
	}

	b, err := RunBench(*duration, *speed)
	if err != nil {
		fmt.Fprintf(os.Stderr, "对照实验失败：%v\n", err)
		os.Exit(1)
	}
	Report(os.Stdout, b)
	fmt.Println()
	fmt.Println("读法：")
	fmt.Println("  1. 计数器看 SAMPLE 各间隔那一列 —— 间隔越大，观测峰值越接近平均值")
	fmt.Println("  2. ON_CHANGE 用在计数器上是错的：值不变就不推，平稳期连样本都没有")
	fmt.Println("  3. 水位看 Gauge 表 —— 长间隔直接错过 200 KB 的尖峰")
	fmt.Println("  4. 状态看 State 表 —— ON_CHANGE 100% 召回，SAMPLE 采样间隔超过 300 ms 就开始漏")
}

// runSingle 起一个临时 mock 服务，做一次 Get 或一次订阅。
func runSingle(sig *Signal, speed float64, getPath, subPath, mode string, interval, duration time.Duration) {
	srv := newLocalServer(NewServer(sig, speed).Handler())
	defer srv.Close()

	if getPath != "" {
		resp, err := http.Get(srv.URL + "/get?path=" + getPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Get 失败：%v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		fmt.Printf("GET %s -> %d\n", getPath, resp.StatusCode)
		buf := make([]byte, 4096)
		n, _ := resp.Body.Read(buf)
		fmt.Println(string(buf[:n]))
		return
	}

	sub := Subscribe(srv.URL, subPath, mode, interval, duration)
	if sub.Err != nil {
		fmt.Fprintf(os.Stderr, "订阅失败：%v\n", sub.Err)
		os.Exit(1)
	}
	fmt.Printf("订阅 %s（mode=%s interval=%s）收到 %d 条更新，%d 字节\n",
		subPath, mode, interval, len(sub.Updates), sub.Bytes)
	for i, u := range sub.Updates {
		if i >= 8 {
			fmt.Println("  ...")
			break
		}
		if u.Kind == "state" {
			fmt.Printf("  t=%6dms  %-10s %s\n", u.TsMS, u.Str, shortPath(u.Path))
		} else {
			fmt.Printf("  t=%6dms  %-10.0f %s\n", u.TsMS, u.Value, shortPath(u.Path))
		}
	}
}

func shortPath(p string) string {
	if len(p) <= 58 {
		return p
	}
	return "..." + p[len(p)-55:]
}
