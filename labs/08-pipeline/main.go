package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	window := flag.Int("window", DefaultWindowSec, "模拟窗口长度（秒）")
	bufferCap := flag.Int("buffer", DefaultBufferCap, "采集器本地缓冲容量（点）")
	fromMin := flag.Int("from", DangerFromSec/60, "查询窗口起点（分钟）")
	toMin := flag.Int("to", DangerToSec/60, "查询窗口终点（分钟）")
	csv := flag.Bool("csv", false, "把各档数据写到 out/")
	flag.Parse()

	if *window < 60 {
		fmt.Fprintln(os.Stderr, "窗口至少 60 秒")
		os.Exit(2)
	}

	pts := Generate(*window)
	series := make([]Series, 0, len(Tiers))
	for _, tier := range Tiers {
		series = append(series, BuildSeries(tier, pts))
	}

	ing := SimulateIngest(pts, *bufferCap)
	from, to := *fromMin*60, *toMin*60
	Report(os.Stdout, pts, series, ing, from, to)

	if *csv {
		if err := writeCSV("out", series); err != nil {
			fmt.Fprintf(os.Stderr, "写 CSV 失败：%v\n", err)
			os.Exit(1)
		}
		fmt.Println("\n各档数据已写入 out/，可用任意工具画图对照")
	}

	fmt.Println()
	fmt.Println("读法：")
	fmt.Println("  1. 1m(mean) 那行峰值等于均值 —— 520 KB 的短时冲顶被摊平成 213 KB，告警直接灭掉")
	fmt.Println("  2. 1m(max) 峰值与 raw 一致，代价是时间分辨率从秒降到分钟，P99 只能当下界看")
	fmt.Println("  3. 1d(max) 峰值也在，但答不出「哪一分钟冲顶」：容量规划够用，事故复盘不够")
	fmt.Println("  4. 背压那段：丢点集中在积压期，健康窗口完整率 100% —— 面板上看起来一切正常")
}

func writeCSV(dir string, series []Series) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, s := range series {
		name := strings.NewReplacer("(", "-", ")", "").Replace(string(s.Tier))
		path := filepath.Join(dir, name+".csv")
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		var b strings.Builder
		b.WriteString("from_sec,to_sec,count,mean_kb,max_kb,p99_kb\n")
		for _, bk := range s.Buckets {
			fmt.Fprintf(&b, "%d,%d,%d,%.1f,%.1f,%.1f\n", bk.FromSec, bk.ToSec, bk.Count, bk.Mean, bk.Max, bk.P99)
		}
		if _, err := f.WriteString(b.String()); err != nil {
			f.Close()
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
	}
	return nil
}
