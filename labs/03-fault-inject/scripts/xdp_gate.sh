#!/usr/bin/env bash
# xdp_gate.sh — 编译 / 挂载最小 XDP 反压闸门，并驱动它的开关。
#
# 关键点：veth 在 containerlab 节点的网络命名空间里，宿主机看不到。
# 所以编译在宿主做，挂载与卸载都经 nsenter --net 切进节点的 netns 执行。
# 三个 map 用 LIBBPF_PIN_BY_NAME 自动 pin 到 /sys/fs/bpf，此后用 bpftool 直接读写。
#
# 用法（需要 Linux + clang + bpftool + iproute2，Codespaces 里都齐）：
#   ./xdp_gate.sh build
#   ./xdp_gate.sh attach [节点] [接口]   默认 leaf2 eth1
#   ./xdp_gate.sh close                 模拟水线触顶：拦截后续数据帧
#   ./xdp_gate.sh open                  模拟水线回落：恢复放行
#   ./xdp_gate.sh stats                 看拦截数与收到的 pause 帧数
#   ./xdp_gate.sh off [节点] [接口]
set -uo pipefail

cd "$(dirname "$0")" || exit 1

OBJ=$PWD/xdp_pfc.o
LAB=${LAB:-fault-inject}
NODE=${2:-leaf2}
IFACE=${3:-eth1}
NETNS=/var/run/netns/clab-${LAB}-${NODE}

need() {
	command -v "$1" >/dev/null 2>&1 || {
		echo "缺少 $1，先装：$2" >&2
		exit 1
	}
}

in_netns() {
	[ -e "$NETNS" ] || {
		echo "找不到 $NETNS，先跑 make deploy" >&2
		exit 1
	}
	nsenter --net="$NETNS" "$@"
}

case "${1:-}" in
build)
	need clang "clang llvm"
	clang -O2 -g -target bpf -c xdp_pfc.c -o "$OBJ"
	echo "✅ 编译完成：$OBJ"
	;;

attach)
	[ -f "$OBJ" ] || { echo "先跑 ./xdp_gate.sh build" >&2; exit 1; }
	# 旧 pin 与当前定义不一致时 libbpf 会拒绝加载，先清掉
	rm -f /sys/fs/bpf/pfc_gate /sys/fs/bpf/pfc_dropped /sys/fs/bpf/pfc_pause_seen
	in_netns ip link set dev "$IFACE" xdp obj "$OBJ" sec xdp
	echo "✅ 已挂到 $NODE:$IFACE 入方向；map 已 pin 到 /sys/fs/bpf/pfc_*"
	;;

close)
	need bpftool "linux-tools-common"
	bpftool map update pinned /sys/fs/bpf/pfc_gate key 0 0 0 0 value 1 0 0 0
	echo "闸门关闭：后续数据帧在驱动层被丢弃（模拟 XOFF 之后对端停发的观测量）"
	;;

open)
	need bpftool "linux-tools-common"
	bpftool map update pinned /sys/fs/bpf/pfc_gate key 0 0 0 0 value 0 0 0 0
	echo "闸门打开：恢复放行（模拟 XON）"
	;;

stats)
	need bpftool "linux-tools-common"
	echo "被拦截的数据帧数（各 CPU 合计）："
	bpftool map dump pinned /sys/fs/bpf/pfc_dropped
	echo "收到的 0x8808 帧数："
	bpftool map dump pinned /sys/fs/bpf/pfc_pause_seen
	;;

off)
	in_netns ip link set dev "$IFACE" xdp off
	echo "✅ 已从 $NODE:$IFACE 卸载"
	;;

*)
	sed -n '2,18p' "$0"
	exit 2
	;;
esac
