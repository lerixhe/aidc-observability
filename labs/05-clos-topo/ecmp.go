package main

import (
	"math"
	"strconv"
)

// 散列结果的统计。这里只做一件事：把「每条上行分到多少流量」算清楚，
// 再给两个能横向比较的数：最大占比与变异系数。

// LinkStat 是一条上行口上的汇总。
type LinkStat struct {
	Link  string // 形如 leaf1→spine1
	Iface string // leaf 侧接口名
	Flows int
	Bytes float64
	Share float64 // 占该 leaf 上行的比例
}

// LeafStat 是一台 leaf 的等价链路组。
type LeafStat struct {
	Leaf     string
	Links    []LinkStat
	Flows    int
	MaxShare float64 // 最忙那条上行的占比
	Ideal    float64 // 均分时应该是多少，即 1/上行数
	Polar    float64 // MaxShare / Ideal，1 表示完美均分
}

// Result 是一次模拟的全部统计。
type Result struct {
	Policy HashPolicy
	Total  int // 总流数
	Local  int // 同 leaf 内互访，不走上行
	Leaves []LeafStat
	CV     float64 // 全部上行口占比的变异系数
	Worst  float64 // 各 leaf 极化度的最大值
}

// Simulate 把流按指定口径散到上行口，产出统计。
func Simulate(flows []Flow, p HashPolicy) Result {
	ups := Uplinks()
	leafOf := map[string]string{}
	for _, h := range Hosts() {
		leafOf[h.Name] = h.Leaf
	}
	leafByIP := map[uint32]string{}
	for _, h := range Hosts() {
		leafByIP[h.IP] = h.Leaf
	}

	// 每条上行单独统计
	type key struct{ leaf, spine string }
	count := map[key]int{}
	bytes := map[key]float64{}

	res := Result{Policy: p, Total: len(flows)}
	for _, f := range flows {
		srcLeaf := leafByIP[f.Src]
		dstLeaf := leafByIP[f.Dst]
		if srcLeaf == dstLeaf {
			res.Local++
			continue
		}
		// 只有源侧 leaf 的上行参与散列选择；spine 到目的 leaf 是确定性转发
		cands := make([]Uplink, 0, 2)
		for _, u := range ups {
			if u.Leaf == srcLeaf {
				cands = append(cands, u)
			}
		}
		idx := PickLink(FibMultipathHash(f, p), len(cands))
		k := key{cands[idx].Leaf, cands[idx].Spine}
		count[k]++
		bytes[k] += f.Bytes
	}

	var shares []float64
	for _, leaf := range Leaves() {
		ls := LeafStat{Leaf: leaf}
		for _, u := range ups {
			if u.Leaf != leaf {
				continue
			}
			k := key{u.Leaf, u.Spine}
			ls.Links = append(ls.Links, LinkStat{
				Link:  u.Leaf + "→" + u.Spine,
				Iface: u.Iface,
				Flows: count[k],
				Bytes: bytes[k],
			})
			ls.Flows += count[k]
		}
		if ls.Flows == 0 {
			continue
		}
		for i := range ls.Links {
			ls.Links[i].Share = float64(ls.Links[i].Flows) / float64(ls.Flows)
			shares = append(shares, ls.Links[i].Share)
			if ls.Links[i].Share > ls.MaxShare {
				ls.MaxShare = ls.Links[i].Share
			}
		}
		ls.Ideal = 1 / float64(len(ls.Links))
		ls.Polar = ls.MaxShare / ls.Ideal
		if ls.Polar > res.Worst {
			res.Worst = ls.Polar
		}
		res.Leaves = append(res.Leaves, ls)
	}

	res.CV = cv(shares)
	return res
}

// cv 算变异系数：标准差 / 均值。均值恒为 1/链路数，所以它衡量的是「偏离均分多远」。
func cv(xs []float64) float64 {
	if len(xs) < 2 {
		return 0
	}
	var sum float64
	for _, x := range xs {
		sum += x
	}
	m := sum / float64(len(xs))
	if m == 0 {
		return 0
	}
	var sq float64
	for _, x := range xs {
		sq += (x - m) * (x - m)
	}
	return math.Sqrt(sq/float64(len(xs))) / m
}

// Ratio 返回「实际 / 理想」的流量比，用于表格里那个 5:3 的列。
func (ls LeafStat) Ratio() string {
	out := ""
	for i, l := range ls.Links {
		if i > 0 {
			out += ":"
		}
		out += strconv.Itoa(l.Flows)
	}
	return out
}
