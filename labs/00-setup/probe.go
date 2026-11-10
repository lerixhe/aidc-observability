package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Lab 是 labs/ 下一个实验的登记信息。
type Lab struct {
	Dir     string   // 目录名
	Layer   string   // L1 无门槛 / L2 需容器
	Needs   []string // 需要的工具
	Lesson  string   // 对应课号
	Summary string
	Ready   bool // 代码是否已落地
}

// knownLabs 是登记表。labs/ 里出现未登记的目录时，自检会提醒。
var knownLabs = []Lab{
	{"00-setup", "L1", []string{"go"}, "01", "环境自检：这台机器能跑哪些 lab", true},
	{"01-traffic-model", "L1", []string{"go"}, "02", "IDC 小流 vs AllReduce 同步突发", true},
	{"02-pfc-sim", "L1", []string{"go"}, "05,06", "PFC / ECN / DCQCN 参数化队列模型", true},
	{"04-nccl-trace", "L1", []string{"go"}, "08", "从 NCCL 日志还原流量矩阵与 straggler", true},
	{"03-fault-inject", "L2", []string{"go", "docker"}, "07", "tc/netem 复现 HoL 与 incast", false},
	{"05-clos-topo", "L2", []string{"go", "docker", "containerlab"}, "09", "2 spine / 4 leaf 的 ECMP 极化", false},
	{"06-gnmi-mock", "L1", []string{"go"}, "11", "可被 gNMI 客户端订阅的 mock server", false},
	{"08-pipeline", "L1", []string{"go"}, "14", "秒级遥测落盘与查询保真度对比", false},
	{"10-rca", "L1", []string{"go"}, "16", "告警图算法做根因排序", false},
}

// Discovery 是一次环境探测的结果。
type Discovery struct {
	Go        ToolStatus
	Docker    ToolStatus
	Clab      ToolStatus
	Python    ToolStatus
	Labs      []Lab
	Unlisted  []string
	RootDir   string
	GoVersion [3]int
}

// ToolStatus 是单个工具的探测结果。
type ToolStatus struct {
	Name    string
	Version string
	OK      bool
	Detail  string
}

// Discover 探测本机环境与 labs 目录。
func Discover(root string) *Discovery {
	d := &Discovery{RootDir: root}
	d.Go = probeGo()
	if d.Go.OK {
		d.GoVersion = parseSemver(d.Go.Version)
	}
	d.Docker = probeCmd("docker", "Docker", []string{"--version"}, "未安装，或不在 PATH 里")
	d.Clab = probeCmd("containerlab", "containerlab", []string{"version"}, "未安装，L2 lab 需要")
	d.Python = probeCmd("python3", "python3", []string{"--version"}, "未安装，部分脚本需要")

	d.Labs, d.Unlisted = scanLabs(root)
	return d
}

// GoAtLeast 判断 Go 版本是否满足要求。
func (d *Discovery) GoAtLeast(major, minor int) bool {
	v := d.GoVersion
	return v[0] > major || (v[0] == major && v[1] >= minor)
}

// CanRun 判断某个 lab 所需工具是否齐备。
func (d *Discovery) CanRun(lab Lab) (bool, []string) {
	missing := []string{}
	for _, need := range lab.Needs {
		switch need {
		case "go":
			if !d.Go.OK || !d.GoAtLeast(1, 22) {
				missing = append(missing, "go>=1.22")
			}
		case "docker":
			if !d.Docker.OK {
				missing = append(missing, "docker")
			}
		case "containerlab":
			if !d.Clab.OK {
				missing = append(missing, "containerlab")
			}
		case "python3":
			if !d.Python.OK {
				missing = append(missing, "python3")
			}
		}
	}
	return len(missing) == 0, missing
}

// ---- 探测工具 ----

func probeGo() ToolStatus {
	ts := ToolStatus{Name: "go"}
	out, err := runCmd("go", "version")
	if err != nil {
		ts.Detail = "未安装 Go，先装 https://go.dev/dl/"
		return ts
	}
	ts.Version = parseSemver(out).String()
	ts.OK = true
	return ts
}

func probeCmd(bin, name string, args []string, missingHint string) ToolStatus {
	ts := ToolStatus{Name: name}
	out, err := runCmd(bin, args...)
	if err != nil {
		ts.Detail = missingHint
		return ts
	}
	ts.Version = parseSemver(out).String()
	ts.OK = true
	// Docker 特别检查：装了但 daemon 没起来是最常见的坑
	if bin == "docker" {
		if _, err := runCmd("docker", "info", "--format", "{{.ServerVersion}}"); err != nil {
			ts.Detail = "已安装但 daemon 没在跑（macOS 需要先启动 Docker Desktop / OrbStack）"
		}
	}
	return ts
}

func runCmd(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	done := make(chan struct{})
	var out []byte
	var err error
	go func() {
		out, err = cmd.CombinedOutput()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		return "", fmt.Errorf("%s 超时", name)
	}
	return string(out), err
}

// ---- 版本解析 ----

// Semver 是粗粒度的版本三元组。
type Semver [3]int

func (v Semver) String() string {
	if v[2] == 0 && v[1] == 0 && v[0] == 0 {
		return "未知"
	}
	return fmt.Sprintf("%d.%d.%d", v[0], v[1], v[2])
}

// parseSemver 从任意命令输出里捞出第一个 x.y[.z]。
func parseSemver(s string) Semver {
	fields := strings.FieldsFunc(s, func(r rune) bool {
		return (r < '0' || r > '9') && r != '.'
	})
	for _, f := range fields {
		f = strings.Trim(f, ".")
		parts := strings.Split(f, ".")
		if len(parts) < 2 {
			continue
		}
		var v Semver
		ok := true
		for i := 0; i < 3 && i < len(parts); i++ {
			n, err := atoi(parts[i])
			if err != nil {
				ok = false
				break
			}
			v[i] = n
		}
		if ok {
			return v
		}
	}
	return Semver{}
}

func atoi(s string) (int, error) {
	n := 0
	if s == "" {
		return 0, fmt.Errorf("空字符串")
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("非数字")
		}
		n = n*10 + int(r-'0')
	}
	return n, nil
}

// ---- labs 目录扫描 ----

// scanLabs 校验登记表里的目录是否存在，并挑出未登记的目录。
func scanLabs(root string) ([]Lab, []string) {
	labsDir := filepath.Join(root, "labs")
	entries, err := os.ReadDir(labsDir)
	if err != nil {
		return nil, nil
	}

	onDisk := map[string]bool{}
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			onDisk[e.Name()] = true
		}
	}

	var registered []Lab
	registeredSet := map[string]bool{}
	for _, lab := range knownLabs {
		registeredSet[lab.Dir] = true
		// 目录不在、或者没有 go.mod，都算未落地
		if _, err := os.Stat(filepath.Join(labsDir, lab.Dir, "go.mod")); err != nil {
			lab.Ready = false
		}
		registered = append(registered, lab)
	}

	var unlisted []string
	for name := range onDisk {
		if !registeredSet[name] {
			unlisted = append(unlisted, name)
		}
	}
	sort.Strings(unlisted)
	return registered, unlisted
}
