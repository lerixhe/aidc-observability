package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	try := flag.Bool("try", false, "额外对每个可跑的 lab 执行 go vet + go test 冒烟")
	rootFlag := flag.String("root", "", "仓库根目录，默认从当前目录向上找含 labs/ 的目录")
	flag.Parse()

	root := *rootFlag
	if root == "" {
		r, err := findRoot()
		if err != nil {
			fmt.Fprintf(os.Stderr, "找不到仓库根目录（向上查找含 labs/ 的目录失败）：%v\n", err)
			os.Exit(1)
		}
		root = r
	}

	d := Discover(root)
	Report(os.Stdout, d)

	if *try {
		fmt.Fprintln(os.Stdout)
		if !SmokeTest(os.Stdout, d) {
			os.Exit(1)
		}
	}

	if !d.GoAtLeast(1, 22) {
		os.Exit(1)
	}
}

// findRoot 从当前目录往上找含 labs/ 子目录的位置。
func findRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if info, err := os.Stat(filepath.Join(dir, "labs")); err == nil && info.IsDir() {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("已到根目录 %s", dir)
		}
		dir = parent
	}
}

// SmokeTest 对每个可跑的 lab 跑 go vet + go test，返回是否全部通过。
func SmokeTest(w io.Writer, d *Discovery) bool {
	fmt.Fprintln(w, "冒烟检查（go vet + go test，每个 lab 最多 120 s）")
	fmt.Println(repeatChar('-', 66))
	fmt.Printf("%-18s %-8s %-6s %-6s %s\n", "lab", "vet", "test", "耗时", "备注")

	allOK := true
	for _, lab := range d.Labs {
		if !lab.Ready || lab.Layer != "L1" {
			continue
		}
		ok, _ := d.CanRun(lab)
		if !ok {
			continue
		}
		dir := filepath.Join(d.RootDir, "labs", lab.Dir)
		start := time.Now()
		vetErr := runIn(dir, "go", "vet", "./...")
		testErr := runIn(dir, "go", "test", "./...")
		elapsed := time.Since(start)

		vetMark, testMark, note := "✅", "✅", ""
		if vetErr != nil {
			vetMark = "❌"
			note = trimErr(vetErr)
			allOK = false
		}
		if testErr != nil {
			testMark = "❌"
			if note != "" {
				note += " / "
			}
			note += trimErr(testErr)
			allOK = false
		}
		fmt.Printf("%-18s %-8s %-6s %-6s %s\n", lab.Dir, vetMark, testMark, elapsed.Round(time.Millisecond), note)
	}
	if allOK {
		fmt.Println("\n全部通过。下一步：cd labs/02-pfc-sim && make compare")
	}
	return allOK
}

func runIn(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s 失败：%s", name, strings.Join(args, " "), firstLine(string(out)))
	}
	return nil
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	if s == "" {
		return "无输出"
	}
	return s
}

func trimErr(err error) string {
	s := err.Error()
	if len(s) > 78 {
		s = s[:75] + "..."
	}
	return s
}
