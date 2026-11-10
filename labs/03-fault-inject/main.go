// 03-fault-inject · 四大故障的注入剧本与遥测指纹
//
// 磁盘上不该有没有活干的日子。这个 lab 干两件事：
//  1. 给出四类故障的注入剧本（容器拓扑能真跑，真机版给了等价做法）
//  2. 把每类故障的遥测形态写成可计算的指纹，并给出一个判读器
//
// 默认模式不碰 Docker：生成剧本 + 生成样本 + 给出判读，全程零依赖。
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	scenario := flag.String("scenario", "all", "场景：storm / deadlock / incast / hol / all")
	planFlag := flag.Bool("plan", false, "只打印注入计划")
	simFlag := flag.Bool("sim", false, "生成样本 CSV 到 -dir，并打印摘要")
	classifyFlag := flag.Bool("classify", false, "判读 -dir 下的样本")
	applyFlag := flag.Bool("apply", false, "执行注入（需 Docker + containerlab 已 deploy）")
	destroyFlag := flag.Bool("destroy", false, "清理全部 qdisc（需 Docker）")
	dir := flag.String("dir", "samples", "样本目录")
	seed := flag.Int64("seed", 42, "随机种子，改它可看抖动对判读的影响")
	flag.Parse()

	th := DefaultThresholds()

	if *destroyFlag {
		if err := Destroy(os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	if *classifyFlag {
		if err := runClassify(*dir, th); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	list, err := pickScenarios(*scenario)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	fail := false
	for _, s := range list {
		s.Seed = *seed
		rows := Simulate(s)

		switch {
		case *planFlag:
			ReportPlan(os.Stdout, s)

		case *simFlag:
			path, err := SaveSample(*dir, s, rows)
			if err != nil {
				fmt.Fprintf(os.Stderr, "写样本失败：%v\n", err)
				os.Exit(1)
			}
			fmt.Printf("样本已写入 %s\n", path)
			ReportSample(os.Stdout, s, rows)

		case *applyFlag:
			ReportPlan(os.Stdout, s)
			fmt.Println("== 开始注入 ==")
			if err := Apply(os.Stdout, s); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			fmt.Println("注入完成。采样后跑：go run . -classify -dir <你的样本目录>")

		default:
			ReportPlan(os.Stdout, s)
			ReportSample(os.Stdout, s, rows)
			v := Classify(rows, th)
			ReportVerdict(os.Stdout, v)
			if msg := ExplainMissing(v.Kind, s.Kind); msg != "" {
				fmt.Fprintf(os.Stdout, "⚠ %s\n\n", msg)
				fail = true
			}
			fmt.Println("----")
			fmt.Println()
		}
	}

	if fail {
		os.Exit(1)
	}
}

// pickScenarios 解析 -scenario，all 表示全部。
func pickScenarios(name string) ([]Scenario, error) {
	if name == "all" || name == "" {
		return Scenarios(), nil
	}
	s, ok := FindScenario(name)
	if !ok {
		return nil, fmt.Errorf("没有场景 %q。可选：storm、deadlock、incast、hol、all", name)
	}
	return []Scenario{s}, nil
}

// runClassify 判读目录下所有样本，输出顺序按文件名，保证 expected 稳定。
func runClassify(dir string, th Thresholds) error {
	files, err := SampleIndex(dir)
	if err != nil {
		return fmt.Errorf("读样本目录 %s 失败：%w。先跑 go run . -sim 生成样本", dir, err)
	}
	if len(files) == 0 {
		return fmt.Errorf("%s 下没有 CSV 样本，先跑 go run . -sim", dir)
	}

	bad := 0
	for _, name := range files {
		path := filepath.Join(dir, name)
		rows, err := LoadSample(path)
		if err != nil {
			return err
		}
		v := Classify(rows, th)
		fmt.Printf("样本 %s（%d 行）\n", name, len(rows))
		ReportVerdict(os.Stdout, v)

		want := scenarioFromSampleName(name)
		if msg := ExplainMissing(v.Kind, want); msg != "" {
			fmt.Printf("⚠ %s\n\n", msg)
			bad++
		}
	}
	if bad > 0 {
		return fmt.Errorf("%d 份样本的判读结论与文件名不符", bad)
	}
	fmt.Println("全部样本判读与场景一致。")
	return nil
}
