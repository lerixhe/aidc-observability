//go:build ignore
// +build ignore

/*
xdp_pfc.c — 最小 XDP 反压闸门（挂容器 veth 入方向）。

队列水线触顶时由用户态把 gate 置 1，驱动层直接拦掉后续数据帧，
等价于「对端被 pause 压住、报文发不过来」；水线回落置 0 即恢复。
配合 scripts/inject_pfc_pause.py 注入真实 0x8808 帧，线上就能抓包对账。

三个 pinned map（用 bpftool 读写）：
  pfc_gate        array[1] u32   1=拦截，0=放行
  pfc_dropped     percpu u64     被拦掉的数据帧数
  pfc_pause_seen  percpu u64     收到的 0x8808 帧数

编译：clang -O2 -g -target bpf -c xdp_pfc.c -o xdp_pfc.o
驱动：./xdp_gate.sh attach|close|open|stats|off
*/

#include <linux/bpf.h>
#include <linux/if_ether.h>
#include <bpf/bpf_helpers.h>

#define ETH_P_PAUSE 0x8808

#define MAP_ARRAY(name, key_t, val_t)                     \
	struct {                                             \
		__uint(type, BPF_MAP_TYPE_ARRAY);              \
		__uint(max_entries, 1);                        \
		__type(key, key_t);                            \
		__type(value, val_t);                          \
		__uint(pinning, LIBBPF_PIN_BY_NAME);           \
	} name SEC(".maps")

#define MAP_CPU(name, key_t, val_t)                       \
	struct {                                             \
		__uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);       \
		__uint(max_entries, 1);                        \
		__type(key, key_t);                            \
		__type(value, val_t);                          \
		__uint(pinning, LIBBPF_PIN_BY_NAME);           \
	} name SEC(".maps")

MAP_ARRAY(pfc_gate, __u32, __u32);
MAP_CPU(pfc_dropped, __u32, __u64);
MAP_CPU(pfc_pause_seen, __u32, __u64);

static __always_inline void bump(void *map)
{
	__u32 key = 0;
	__u64 *v = bpf_map_lookup_elem(map, &key);

	if (v)
		__sync_fetch_and_add(v, 1);
}

SEC("xdp")
int xdp_pfc_gate(struct xdp_md *ctx)
{
	void *data = (void *)(long)ctx->data;
	void *data_end = (void *)(long)ctx->data_end;
	struct ethhdr *eth = data;
	__u32 key = 0;
	__u32 *gate;

	if ((void *)(eth + 1) > data_end)
		return XDP_PASS;

	/* 流控帧本身必须放行，否则注入端会被自己的闸门吃掉 */
	if (eth->h_proto == __constant_htons(ETH_P_PAUSE)) {
		bump(&pfc_pause_seen);
		return XDP_PASS;
	}

	gate = bpf_map_lookup_elem(&pfc_gate, &key);
	if (gate && *gate) {
		bump(&pfc_dropped);
		return XDP_DROP;
	}

	return XDP_PASS;
}

char _license[] SEC("license") = "GPL";
