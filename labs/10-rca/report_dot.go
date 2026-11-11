package main

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// ReportDot 导出 Graphviz DOT 格式的拓扑与故障传播因果图。
// 节点按 RCA 置信度打分着色（红/橙/灰），并标出真实故障点。
func ReportDot(w io.Writer, topo *Topo, cands []Candidate, truth Truth) {
	scoreMap := map[string]Candidate{}
	for _, c := range cands {
		scoreMap[c.Entity] = c
	}

	fmt.Fprintln(w, "digraph AIDC_RCA {")
	fmt.Fprintln(w, "  rankdir=TB;")
	fmt.Fprintln(w, "  node [shape=record, style=\"filled,rounded\", fontname=\"Helvetica\", fontsize=10];")
	fmt.Fprintln(w, "  edge [fontname=\"Helvetica\", fontsize=8, color=\"#718096\"];")
	fmt.Fprintln(w)

	// 按层级子图聚类排布
	for _, tier := range []Tier{TierSpine, TierLeaf, TierHost} {
		fmt.Fprintf(w, "  subgraph cluster_%s {\n", tier)
		fmt.Fprintf(w, "    label=\"%s Tier\";\n", strings.Title(tier.String()))
		fmt.Fprintln(w, "    style=\"dashed\"; color=\"#cbd5e0\";")

		var nodesInTier []string
		for _, n := range topo.Nodes {
			if n.Tier == tier {
				nodesInTier = append(nodesInTier, n.Name)
			}
		}
		sort.Strings(nodesInTier)

		for _, name := range nodesInTier {
			cand, hasCand := scoreMap[name]
			fillColor := "#edf2f7"
			fontColor := "#2d3748"
			borderWidth := "1"
			borderColor := "#a0aec0"

			label := name
			if hasCand && cand.Score > 0 {
				label = fmt.Sprintf("%s | score: %.2f\\n(cov:%.0f%%, prec:%.0f%%)",
					name, cand.Score, cand.Coverage*100, cand.Precision*100)
				if cand.Score >= 0.7 {
					fillColor = "#fed7d7" // 浅红高危
					borderColor = "#e53e3e"
					borderWidth = "2"
				} else if cand.Score >= 0.4 {
					fillColor = "#feebc8" // 浅橙中危
					borderColor = "#dd6b20"
				}
			}

			if truth.Entity == name {
				label = fmt.Sprintf("★ [TRUTH ROOT] ★\\n%s", label)
				borderColor = "#c53030"
				borderWidth = "3"
			}

			fmt.Fprintf(w, "    \"%s\" [label=\"{%s}\", fillcolor=\"%s\", fontcolor=\"%s\", color=\"%s\", penwidth=%s];\n",
				name, label, fillColor, fontColor, borderColor, borderWidth)
		}
		fmt.Fprintln(w, "  }")
		fmt.Fprintln(w)
	}

	// 拓扑有向边（Spine -> Leaf -> Host，故障向下流）
	for _, n := range topo.Nodes {
		if n.Tier == TierSpine {
			for _, m := range topo.Adj[n.Name] {
				if topo.TierOf(m) == TierLeaf {
					fmt.Fprintf(w, "  \"%s\" -> \"%s\";\n", n.Name, m)
				}
			}
		} else if n.Tier == TierLeaf {
			for _, m := range topo.Adj[n.Name] {
				if topo.TierOf(m) == TierHost {
					fmt.Fprintf(w, "  \"%s\" -> \"%s\";\n", n.Name, m)
				}
			}
		}
	}

	fmt.Fprintln(w, "}")
}
