package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	days := flag.Int("days", DefaultDays, "模拟窗口天数")
	slo := flag.Float64("slo", SLOTarget, "目标可用率，默认 0.999")
	flag.Parse()

	if *days < 1 || *days > 365 {
		fmt.Fprintln(os.Stderr, "days 取值范围 1..365")
		os.Exit(2)
	}
	if *slo <= 0 || *slo >= 1 {
		fmt.Fprintln(os.Stderr, "slo 必须在 (0,1) 之间，例如 0.999")
		os.Exit(2)
	}

	samples := Generate(*days)
	results := make([]Result, 0, len(Calibers))
	for _, c := range Calibers {
		results = append(results, Compute(c, samples, *slo))
	}

	timeB := TimeBudget(samples, *slo)
	eventB := EventBudget(samples, *slo)
	burns, fastPage := MultiWindowBurn(samples, *slo)

	Report(os.Stdout, *days, samples, results, timeB, eventB, burns, fastPage, *slo)

	fmt.Println()
	fmt.Println("读法：")
	fmt.Println("  1. 三种口径能给出「一个达标、两个未达标」—— 说「可用率 99.9%」而不说口径，等于没说")
	fmt.Println("  2. 事件口径最宽：夜间一小时的劣化只摊掉 0.04% 的请求，时间口径上却是 0.14% 的时间")
	fmt.Println("  3. 分位口径最严也最瞎：只看时延，失败快也算「达标」")
	fmt.Println("  4. 燃烧率要多窗口成对看：单看 5m 会被一次抖动打爆，单看 1h 会漏掉刚开始的事故")
}
