package main

import "fmt"

// Tier 表示设备在 CLOS 里的层级。故障沿拓扑向下流，所以层级就是传播方向。
type Tier int

const (
	TierSpine Tier = iota
	TierLeaf
	TierHost
)

func (t Tier) String() string {
	switch t {
	case TierSpine:
		return "spine"
	case TierLeaf:
		return "leaf"
	case TierHost:
		return "host"
	}
	return "?"
}

// Node 是图上的一个节点。
type Node struct {
	Name string
	Tier Tier
}

// Topo 是「图即拓扑」的那张图。
//
// 三个索引各有分工：
//
//	Down  下游集合（含自身），RCA 用它算「一个候选能解释多少告警」
//	Adj   无向邻接，告警收敛时用来判断两个实体是否同源
//	Tier  层级，注入故障时决定告警怎么往下传播
type Topo struct {
	Nodes  []Node
	byName map[string]Node
	Down   map[string][]string
	Adj    map[string][]string
}

// BuildClos 建一张 spines × leaves × hostsPerLeaf 的 CLOS 图。
// 每个 leaf 双上行到所有 spine，这是智算集群的常规接法。
func BuildClos(spines, leaves, hostsPerLeaf int) *Topo {
	t := &Topo{
		byName: map[string]Node{},
		Down:   map[string][]string{},
		Adj:    map[string][]string{},
	}
	add := func(name string, tier Tier) {
		t.Nodes = append(t.Nodes, Node{Name: name, Tier: tier})
		t.byName[name] = Node{Name: name, Tier: tier}
		t.Adj[name] = nil
	}

	for s := 0; s < spines; s++ {
		add(fmt.Sprintf("spine%d", s), TierSpine)
	}
	for l := 0; l < leaves; l++ {
		add(fmt.Sprintf("leaf%d", l), TierLeaf)
		for h := 0; h < hostsPerLeaf; h++ {
			add(fmt.Sprintf("host%d", l*hostsPerLeaf+h), TierHost)
		}
	}

	for s := 0; s < spines; s++ {
		for l := 0; l < leaves; l++ {
			link(t, fmt.Sprintf("spine%d", s), fmt.Sprintf("leaf%d", l))
		}
	}
	for l := 0; l < leaves; l++ {
		for h := 0; h < hostsPerLeaf; h++ {
			link(t, fmt.Sprintf("leaf%d", l), fmt.Sprintf("host%d", l*hostsPerLeaf+h))
		}
	}

	t.buildDown()
	return t
}

func link(t *Topo, a, b string) {
	t.Adj[a] = append(t.Adj[a], b)
	t.Adj[b] = append(t.Adj[b], a)
}

// buildDown 按层级关系算出每个节点的下游集合（含自身）。
func (t *Topo) buildDown() {
	for _, n := range t.Nodes {
		set := []string{n.Name}
		if n.Tier != TierHost {
			for _, m := range t.Nodes {
				if m.Tier <= n.Tier {
					continue // 只看严格下游
				}
				if t.reachable(n.Name, m.Name) {
					set = append(set, m.Name)
				}
			}
		}
		t.Down[n.Name] = set
	}
}

// reachable 判断 from 能否沿下行到达 to（BFS 带上层约束）。
func (t *Topo) reachable(from, to string) bool {
	fTier := t.byName[from].Tier
	tTier := t.byName[to].Tier
	if fTier >= tTier {
		return false
	}
	// CLOS 结构里上层到下层只要存在路径即算可达
	visited := map[string]bool{from: true}
	queue := []string{from}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, nb := range t.Adj[cur] {
			if t.byName[nb].Tier <= fTier {
				continue // 不往上走
			}
			if nb == to {
				return true
			}
			if !visited[nb] {
				visited[nb] = true
				queue = append(queue, nb)
			}
		}
	}
	return false
}

// Names 返回所有节点名，按层级与序号的稳定顺序。
func (t *Topo) Names() []string {
	out := make([]string, len(t.Nodes))
	for i, n := range t.Nodes {
		out[i] = n.Name
	}
	return out
}

// TierOf 返回节点层级。
func (t *Topo) TierOf(name string) Tier {
	return t.byName[name].Tier
}

// Contains 判断下游集合里是否包含某节点。
func (t *Topo) Contains(root, node string) bool {
	for _, v := range t.Down[root] {
		if v == node {
			return true
		}
	}
	return false
}

// IsAdjacent 判断两个节点是否有直连边。
func (t *Topo) IsAdjacent(a, b string) bool {
	for _, v := range t.Adj[a] {
		if v == b {
			return true
		}
	}
	return false
}
