#!/usr/bin/env bash
# 采 leaf 上行的字节计数，算真实分布，用来和 Go 里的散列模型对照。
#
# 用法：./scripts/collect.sh [采样秒数] [流数]
#   采样秒数默认 10 秒
#   流数默认 8，脚本会用 iperf3 起这么多并发流
#
# 输出：out/live-ecmp.csv，列与判读逻辑无关，只给人看
#
# 需要 containerlab 已经 deploy（make deploy）。

set -euo pipefail

DUR="${1:-10}"
FLOWS="${2:-8}"
PREFIX="clab-clos-topo-"
OUT="out/live-ecmp.csv"

if ! command -v docker >/dev/null 2>&1; then
  echo "需要 docker。没有 Docker 时用 go run . 看模型结论即可。" >&2
  exit 1
fi

if ! docker ps --format '{{.Names}}' | grep -q "^${PREFIX}leaf1$"; then
  echo "拓扑没起来。先跑：make deploy" >&2
  exit 1
fi

echo "起 $FLOWS 条流：h1 → h7（跨 leaf，必然走上行）"
docker exec -d "${PREFIX}h7" iperf3 -s -D >/dev/null
sleep 1
for i in $(seq 1 "$FLOWS"); do
  docker exec -d "${PREFIX}h1" iperf3 -c 10.11.4.10 -t $((DUR + 10)) -b 50M -P 1 -p 5201 || true
done

# 计数是单调递增的，采两次取差值，才看得到「这段时间里谁在跑」
declare -A before
for leaf in leaf1 leaf2 leaf3 leaf4; do
  for iface in eth3 eth4; do
    before["${leaf}:${iface}"]=$(docker exec "${PREFIX}${leaf}" cat "/sys/class/net/${iface}/statistics/tx_bytes")
  done
done

echo "等 ${DUR} 秒，让流量跑起来..."
sleep "$DUR"

declare -A after
for leaf in leaf1 leaf2 leaf3 leaf4; do
  for iface in eth3 eth4; do
    after["${leaf}:${iface}"]=$(docker exec "${PREFIX}${leaf}" cat "/sys/class/net/${iface}/statistics/tx_bytes")
  done
done

mkdir -p out
echo "leaf,iface,spine,bytes,share" > "$OUT"

# 先算每条 leaf 的上下行总量，再算占比：跨 leaf 的流只从源侧 leaf 出去
for leaf in leaf1 leaf2 leaf3 leaf4; do
  total=0
  for iface in eth3 eth4; do
    d=$(( ${after["${leaf}:${iface}"]} - ${before["${leaf}:${iface}"]} ))
    total=$(( total + d ))
  done
  for iface in eth3 eth4; do
    d=$(( ${after["${leaf}:${iface}"]} - ${before["${leaf}:${iface}"]} ))
    spine="spine1"
    [ "$iface" = "eth4" ] && spine="spine2"
    if [ "$total" -gt 0 ]; then
      share=$(awk -v d="$d" -v t="$total" 'BEGIN{printf "%.3f", d/t}')
    else
      share="0.000"
    fi
    echo "${leaf},${iface},${spine},${d},${share}" >> "$OUT"
  done
done

echo
echo "采样结果（${DUR} 秒窗口）："
column -s, -t "$OUT"
echo
echo "判读：两条上行的 share 越接近 0.5 越均匀。"
echo "     某台 leaf 上出现 0.9/0.1 这种分布，就是极化，与 Go 模型里"
echo "     流数少时的结论一致。"
echo
echo "自动清理流量："
for i in $(seq 1 "$FLOWS"); do
  docker exec "${PREFIX}h1" pkill -f "iperf3 -c" 2>/dev/null || true
done
