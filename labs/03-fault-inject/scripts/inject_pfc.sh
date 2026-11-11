#!/usr/bin/env bash
# inject_pfc.sh — 向 containerlab 节点的 veth 注入真实 0x8808 PFC pause 帧。
#
# 两条路径，优先容器内 python3：
#   1) 容器里有 python3（netshoot 一般有）→ docker cp 进容器直接发
#   2) 没有 → 用宿主机的 python3 配合 nsenter 切进容器网络命名空间发
#
# 用法：
#   ./inject_pfc.sh [节点] [接口] [优先级] [帧数]
#   ./inject_pfc.sh leaf2 eth1 3 10
#
# 验证：
#   docker exec clab-fault-inject-h3 tcpdump -i eth1 ether proto 0x8808 -c 3 -xx
set -uo pipefail

cd "$(dirname "$0")" || exit 1

LAB=${LAB:-fault-inject}
NODE=${1:-leaf2}
IFACE=${2:-eth1}
PRIO=${3:-3}
COUNT=${4:-10}

CTR="clab-${LAB}-${NODE}"

if ! command -v docker >/dev/null 2>&1; then
	echo "找不到 docker；注入只能在 Docker + containerlab 环境里跑（推荐 Codespaces）" >&2
	exit 1
fi
if ! docker inspect "$CTR" >/dev/null 2>&1; then
	echo "容器 $CTR 不存在，先跑：make deploy" >&2
	exit 1
fi

echo "目标容器 $CTR · 接口 $IFACE · 优先级 $PRIO · 帧数 $COUNT"

if docker exec "$CTR" sh -lc 'command -v python3 >/dev/null 2>&1'; then
	echo "路径一：容器内 python3 直接发送"
	docker cp inject_pfc_pause.py "$CTR":/tmp/inject_pfc_pause.py
	docker exec "$CTR" python3 /tmp/inject_pfc_pause.py -i "$IFACE" -p "$PRIO" -c "$COUNT"
	exit $?
fi

# 容器里没有 python3：借助 containerlab 的 netns 挂载点，用宿主机 python3 发
NS="/var/run/netns/${CTR}"
if [ ! -e "$NS" ]; then
	echo "容器里没有 python3，也找不到 netns $NS" >&2
	echo "退路：在容器内装 python3，或直接用 tc 观察反压形态（见 README 第 4 节）" >&2
	exit 1
fi
if ! command -v python3 >/dev/null 2>&1; then
	echo "宿主机也没有 python3" >&2
	exit 1
fi

echo "路径二：宿主机 python3 + nsenter 切进 $NS"
nsenter --net="$NS" python3 "$PWD/inject_pfc_pause.py" -i "$IFACE" -p "$PRIO" -c "$COUNT"
