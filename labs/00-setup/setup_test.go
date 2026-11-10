package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseSemver(t *testing.T) {
	cases := map[string]Semver{
		"go version go1.26.3 darwin/arm64":    {1, 26, 3},
		"Docker version 27.3.1, build abcdef": {27, 3, 1},
		"Python 3.14.7":                       {3, 14, 7},
		"version: 0.59.0\n":                   {0, 59, 0},
		"":                                    {},
		"没有版本号的输出":                            {},
	}
	for in, want := range cases {
		if got := parseSemver(in); got != want {
			t.Errorf("parseSemver(%q) = %v，期望 %v", in, got, want)
		}
	}
}

func TestParseSemverRejectsZeroMajorOnly(t *testing.T) {
	// 只有两段的 0.59 也要解析出来，否则 containerlab 版本显示不出来
	if got := parseSemver("version: 0.59"); got != (Semver{0, 59, 0}) {
		t.Errorf("得到 %v", got)
	}
}

func TestGoAtLeast(t *testing.T) {
	d := &Discovery{GoVersion: [3]int{1, 26, 3}}
	if !d.GoAtLeast(1, 22) {
		t.Error("1.26.3 应满足 >= 1.22")
	}
	if d.GoAtLeast(1, 27) {
		t.Error("1.26.3 不应满足 >= 1.27")
	}
	if d.GoAtLeast(2, 0) {
		t.Error("1.26.3 不应满足 >= 2.0")
	}
}

func TestCanRunReportsMissingTools(t *testing.T) {
	d := &Discovery{
		Go: ToolStatus{Name: "go", OK: true,
			Version: "1.26.3"},
		GoVersion: [3]int{1, 26, 3},
		Docker:    ToolStatus{Name: "docker", OK: false},
		Clab:      ToolStatus{Name: "containerlab", OK: false},
	}

	if ok, missing := d.CanRun(Lab{Dir: "x", Needs: []string{"go"}}); !ok {
		t.Errorf("只有 go 依赖却报缺 %v", missing)
	}
	if ok, missing := d.CanRun(Lab{Dir: "y", Needs: []string{"go", "docker"}}); ok || len(missing) != 1 {
		t.Errorf("缺 docker 应报不可跑，实际 ok=%v missing=%v", ok, missing)
	}
	if ok, missing := d.CanRun(Lab{Dir: "z", Needs: []string{"containerlab"}}); ok ||
		len(missing) != 1 || missing[0] != "containerlab" {
		t.Errorf("缺 containerlab 应报不可跑，实际 ok=%v missing=%v", ok, missing)
	}
}

func TestInterpreterTooOld(t *testing.T) {
	d := &Discovery{Go: ToolStatus{OK: true, Version: "1.20.1"}, GoVersion: [3]int{1, 20, 1}}
	if ok, _ := d.CanRun(Lab{Needs: []string{"go"}}); ok {
		t.Error("Go 1.20 不应满足 >= 1.22")
	}
}

func TestScanLabsPicksUpUnlistedDir(t *testing.T) {
	root := t.TempDir()
	labs := filepath.Join(root, "labs")
	for _, dir := range []string{"02-pfc-sim", "99-experimental"} {
		if err := os.MkdirAll(filepath.Join(labs, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// 只有 02-pfc-sim 有 go.mod，99-experimental 是未登记目录
	if err := os.WriteFile(filepath.Join(labs, "02-pfc-sim", "go.mod"), []byte("module x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	registered, unlisted := scanLabs(root)
	if len(unlisted) != 1 || unlisted[0] != "99-experimental" {
		t.Errorf("未登记目录 = %v，期望 [99-experimental]", unlisted)
	}

	found := false
	for _, lab := range registered {
		if lab.Dir == "02-pfc-sim" {
			found = true
			if !lab.Ready {
				t.Error("有 go.mod 的 lab 应算已落地")
			}
		}
	}
	if !found {
		t.Error("没有把 02-pfc-sim 纳入盘点")
	}
}

func TestScanLabsMarksMissingGoMod(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "labs", "00-setup"), 0o755); err != nil {
		t.Fatal(err)
	}
	registered, _ := scanLabs(root)
	for _, lab := range registered {
		if lab.Dir == "00-setup" && lab.Ready {
			t.Error("没有 go.mod 的目录不应算已落地")
		}
	}
}

func TestFindRootWalksUp(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "labs"), 0o755); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "labs", "02-pfc-sim")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(old) }()
	if err := os.Chdir(nested); err != nil {
		t.Fatal(err)
	}

	got, err := findRoot()
	if err != nil {
		t.Fatalf("findRoot 失败：%v", err)
	}
	// macOS 的 /var 是 /private/var 的软链，用 EvalSymlinks 归一
	wantResolved, _ := filepath.EvalSymlinks(root)
	gotResolved, _ := filepath.EvalSymlinks(got)
	if wantResolved != gotResolved {
		t.Errorf("findRoot = %s，期望 %s", gotResolved, wantResolved)
	}
}
