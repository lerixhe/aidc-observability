package main

import (
	"os"
	"regexp"
	"testing"
)

// 拓扑文件与注入计划是两份手写的文本，最容易出的错是接口名对不上：
// 计划里写 leaf2:eth2，拓扑里到 h3 的口其实是 eth3。这种错只有真跑才会暴露，
// 而真跑需要 Docker。所以在这里先把两份文本对齐。

var linkRe = regexp.MustCompile(`\["([\w-]+):(\w+)",\s*"([\w-]+):(\w+)"\]`)
var devRe = regexp.MustCompile(`\bdev\s+(\w+)`)

// topologyIfaces 从 topo.clab.yml 里解析出「节点 → 接口集合」。
func topologyIfaces(t *testing.T) map[string]map[string]bool {
	t.Helper()
	b, err := os.ReadFile("topo.clab.yml")
	if err != nil {
		t.Fatalf("读不到 topo.clab.yml：%v", err)
	}
	nodes := map[string]map[string]bool{}
	for _, m := range linkRe.FindAllStringSubmatch(string(b), -1) {
		for i := 1; i < len(m); i += 2 {
			node, iface := m[i], m[i+1]
			if nodes[node] == nil {
				nodes[node] = map[string]bool{}
			}
			nodes[node][iface] = true
		}
	}
	if len(nodes) == 0 {
		t.Fatal("topo.clab.yml 里没解析出任何链路端点，格式可能变了")
	}
	return nodes
}

func TestTopologyHasAllNodesInPlan(t *testing.T) {
	nodes := topologyIfaces(t)
	for _, s := range Scenarios() {
		for _, st := range s.Injection {
			if nodes[st.Node] == nil {
				t.Errorf("场景 %s 指向的节点 %s 在拓扑里没有链路，容器建出来连不通", s.Name, st.Node)
			}
		}
	}
}

func TestPlanInterfacesExist(t *testing.T) {
	nodes := topologyIfaces(t)
	for _, s := range Scenarios() {
		for _, st := range s.Injection {
			for _, m := range devRe.FindAllStringSubmatch(st.Cmd, -1) {
				iface := m[1]
				if !nodes[st.Node][iface] {
					t.Errorf("场景 %s 在 %s 上操作 %s，但拓扑里该节点没有这个接口（有 %v）",
						s.Name, st.Node, iface, keysOf(nodes[st.Node]))
				}
			}
		}
	}
}

func keysOf(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
