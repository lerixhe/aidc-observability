package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// 本文件负责两件与「真跑」相关的事：
//   1. 把 dry-run 打出来的命令真正执行掉（需要 Docker + containerlab 已经 deploy）
//   2. 读回真机采集的样本，交给判读器
//
// 默认路径不碰这两样东西，所以没有 Docker 也能跑通全部结论。

// ContainerPrefix 是 containerlab 的容器命名规则：clab-<拓扑名>-<节点名>。
const ContainerPrefix = "clab-fault-inject-"

// containerName 把拓扑里的逻辑节点名翻译成容器名。
func containerName(node string) string {
	return ContainerPrefix + node
}

// Apply 执行一个场景的注入计划。需要拓扑已经 deploy 完成。
func Apply(w io.Writer, s Scenario) error {
	if _, err := exec.LookPath("docker"); err != nil {
		return fmt.Errorf("找不到 docker：%w。注入只能在装有 Docker 的 Linux 或 Codespaces 里跑，"+
			"没有 Docker 时用 -plan 看剧本即可", err)
	}
	for i, st := range s.Injection {
		name := containerName(st.Node)
		fmt.Fprintf(w, "[%d/%d] %s · %s\n      $ %s\n", i+1, len(s.Injection), name, st.Note, st.Cmd)
		cmd := exec.Command("docker", "exec", name, "sh", "-lc", st.Cmd)
		cmd.Stdout = w
		cmd.Stderr = w
		if err := cmd.Run(); err != nil {
			// 注入命令里有些是「持续观察型」（tc -s、ss -tin），失败不影响实验，
			// 但要让读者看见，所以打印而不是中断。
			fmt.Fprintf(w, "      ! 该步返回错误：%v（若命令是观察类可忽略）\n", err)
		}
	}
	return nil
}

// Destroy 清掉场景挂过的 qdisc，把拓扑恢复成干净状态。
func Destroy(w io.Writer) error {
	nodes := []struct{ node, iface string }{
		{"leaf1", "eth1"}, {"leaf1", "eth2"}, {"leaf1", "eth3"},
		{"leaf2", "eth1"}, {"leaf2", "eth2"}, {"leaf2", "eth3"},
		{"h1", "eth1"}, {"h2", "eth1"}, {"h3", "eth1"}, {"h4", "eth1"},
	}
	for _, n := range nodes {
		cmd := exec.Command("docker", "exec", containerName(n.node), "sh", "-lc",
			fmt.Sprintf("tc qdisc del dev %s root 2>/dev/null || true", n.iface))
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(w, "  %s:%s 清理失败：%v\n", n.node, n.iface, err)
			continue
		}
		fmt.Fprintf(w, "  %s:%s 已清理\n", n.node, n.iface)
	}
	return nil
}

// LoadSample 读回一份样本 CSV。真机采集与模型生成用同一个格式。
func LoadSample(path string) ([]Row, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	head, err := r.Read()
	if err != nil {
		return nil, err
	}
	idx := make(map[string]int, len(head))
	for i, h := range head {
		idx[strings.TrimSpace(h)] = i
	}
	need := []string{"port", "role", "t_s", "lat_p50_ms", "lat_p99_ms", "loss_pct", "pause_pct", "ecn_pct", "qdepth_kb"}
	for _, n := range need {
		if _, ok := idx[n]; !ok {
			return nil, fmt.Errorf("%s 缺少列 %q，需要的列见 samples/model-storm.csv 表头", path, n)
		}
	}

	get := func(rec []string, name string) string {
		i := idx[name]
		if i < len(rec) {
			return strings.TrimSpace(rec[i])
		}
		return ""
	}
	num := func(rec []string, name string) float64 {
		v, _ := strconv.ParseFloat(get(rec, name), 64)
		return v
	}

	var rows []Row
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		rows = append(rows, Row{
			Port:      get(rec, "port"),
			Role:      get(rec, "role"),
			T:         num(rec, "t_s"),
			LatP50:    num(rec, "lat_p50_ms"),
			LatP99:    num(rec, "lat_p99_ms"),
			LossPct:   num(rec, "loss_pct"),
			PausePct:  num(rec, "pause_pct"),
			ECNPct:    num(rec, "ecn_pct"),
			QDepthKB:  num(rec, "qdepth_kb"),
			CRCPerMin: num(rec, "crc_per_min"),
			FECBER:    num(rec, "fec_ber"),
		})
	}
	return rows, nil
}

// SampleIndex 找出一个目录下的全部样本文件，按名字排序保证输出稳定。
func SampleIndex(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".csv") {
			continue
		}
		out = append(out, e.Name())
	}
	return out, nil
}

// scenarioFromSampleName 从 model-storm.csv 这种名字反推场景名。
func scenarioFromSampleName(name string) string {
	name = strings.TrimSuffix(name, ".csv")
	name = strings.TrimPrefix(name, "model-")
	return name
}

// WriteCollectScript 生成真机采集脚本的提示，告诉读者该采什么。
func WriteCollectScript(w *bufio.Writer) {
	fmt.Fprintln(w, "真机采集：跑完注入后，按下表把计数器抄进 CSV，列名与 samples/ 保持一致。")
	fmt.Fprintln(w, "  port      端口标识，如 leaf2:eth1")
	fmt.Fprintln(w, "  role      culprit / victim / healthy，仅用于对照，判读不使用")
	fmt.Fprintln(w, "  t_s        相对时间，秒")
	fmt.Fprintln(w, "  lat_p99_ms 路径探测的 P99 时延，超时填 -1")
	fmt.Fprintln(w, "  loss_pct   丢包率")
	fmt.Fprintln(w, "  pause_pct  该周期内处于反压状态的时间占比")
	fmt.Fprintln(w, "  ecn_pct    ECN 标记率")
	fmt.Fprintln(w, "  qdepth_kb  队列水位")
	fmt.Fprintln(w, "  crc_per_min / fec_ber  端口错误计数与 FEC 前误码率")
}
