package main

import (
	"math/rand"
	"net/netip"
	"strconv"
)

// 拓扑与工作负载。写死的部分只有形状：4 leaf × 2 上行到 2 台 spine。
// 变的部分是流：多少条、什么口径、端口固定还是随机。

// Host 是一台挂载在 leaf 下的主机。
type Host struct {
	Name string
	Leaf string
	IP   uint32
}

// Uplink 是 leaf 上的一条等价上行。
type Uplink struct {
	Leaf   string
	Spine  string
	Iface  string // leaf 侧接口名，与 topo.clab.yml 一致
	Local  uint32 // leaf 侧地址
	Remote uint32 // spine 侧地址
}

func mustIPv4(s string) uint32 {
	a := netip.MustParseAddr(s).As4()
	return uint32(a[0])<<24 | uint32(a[1])<<16 | uint32(a[2])<<8 | uint32(a[3])
}

// Hosts 返回八台主机，每台 leaf 挂两台。
func Hosts() []Host {
	return []Host{
		{"h1", "leaf1", mustIPv4("10.11.1.10")},
		{"h2", "leaf1", mustIPv4("10.11.1.11")},
		{"h3", "leaf2", mustIPv4("10.11.2.10")},
		{"h4", "leaf2", mustIPv4("10.11.2.11")},
		{"h5", "leaf3", mustIPv4("10.11.3.10")},
		{"h6", "leaf3", mustIPv4("10.11.3.11")},
		{"h7", "leaf4", mustIPv4("10.11.4.10")},
		{"h8", "leaf4", mustIPv4("10.11.4.11")},
	}
}

// Leaves 返回四台 leaf 的名字，顺序固定。
func Leaves() []string { return []string{"leaf1", "leaf2", "leaf3", "leaf4"} }

// Uplinks 返回八条上行链路：每台 leaf 两条，分别到 spine1 与 spine2。
func Uplinks() []Uplink {
	out := make([]Uplink, 0, 8)
	for i, leaf := range Leaves() {
		out = append(out,
			Uplink{
				Leaf: leaf, Spine: "spine1", Iface: "eth3",
				Local:  mustIPv4("10.12." + strconv.Itoa(i+1) + ".1"),
				Remote: mustIPv4("10.12." + strconv.Itoa(i+1) + ".2"),
			},
			Uplink{
				Leaf: leaf, Spine: "spine2", Iface: "eth4",
				Local:  mustIPv4("10.13." + strconv.Itoa(i+1) + ".1"),
				Remote: mustIPv4("10.13." + strconv.Itoa(i+1) + ".2"),
			},
		)
	}
	return out
}

// Workload 是流集合的生成口径。
type Workload string

const (
	// WorkRandom 随机源目主机与随机源端口，模拟大量短连接
	WorkRandom Workload = "random"
	// WorkElephant 固定源目主机与固定端口，模拟少数同质大象流
	WorkElephant Workload = "elephant"
	// WorkSameTuple 固定源目 IP 但端口递增，用来暴露三层 hash 口径的缺陷
	WorkSameTuple Workload = "sametuple"
)

// Generate 按口径生成 n 条流。固定种子保证可复现。
func Generate(kind Workload, n int, seed int64) []Flow {
	rng := rand.New(rand.NewSource(seed))
	hosts := Hosts()
	out := make([]Flow, 0, n)

	for i := 0; i < n; i++ {
		switch kind {
		case WorkElephant:
			// 同一对主机、同一个端口：数据面看这 n 条流几乎是同一条流
			out = append(out, Flow{
				Src: hosts[0].IP, Dst: hosts[6].IP,
				SPort: 40000, DPort: 5001, Proto: 6, Bytes: 1,
				Label: "h1→h7",
			})
		case WorkSameTuple:
			// IP 对固定，端口逐条变化：三层口径下会全部撞到一条上行
			out = append(out, Flow{
				Src: hosts[0].IP, Dst: hosts[6].IP,
				SPort: uint16(40000 + i%20000), DPort: 5001, Proto: 6, Bytes: 1,
				Label: "h1→h7",
			})
		default:
			// 随机一对主机（跨 leaf 才会走上行，同 leaf 的下一层过滤）
			a := rng.Intn(len(hosts))
			b := rng.Intn(len(hosts) - 1)
			if b >= a {
				b++
			}
			out = append(out, Flow{
				Src: hosts[a].IP, Dst: hosts[b].IP,
				SPort: uint16(32768 + rng.Intn(28000)), DPort: uint16(5001 + rng.Intn(4)),
				Proto: 6, Bytes: 1,
				Label: hosts[a].Name + "→" + hosts[b].Name,
			})
		}
	}
	return out
}
