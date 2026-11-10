package main

// ECMP 的散列复刻。
//
// 数据面怎么写这个 hash 的，决定了两条等价链路会不会一条吃满、一条闲着。
// Linux 的实现分两段：
//
//	IPv4 三层口径：hash = jhash_2words(saddr, daddr, 0)
//	  —— 只看两个 IP。同一对主机的所有连接，全部落同一条链路。
//	IPv4 五元组口径：先混地址，再混端口与协议
//	  —— 端口参与，同一对 IP 的连接可以被端口打散。
//
// 口径由 net.ipv4.fib_multipath_hash_policy 控制：0 走三层，1 走四层。
// 这里复刻的就是这两条路径，用的是 Jenkins lookup3（内核 jhash 同源算法）。
//
// 边界：这是近似复刻。内核拼 keys 的顺序随版本变，绝对值不必对齐；
// 要看的是「哪种口径会把流量堆到一条路上」，分布特性才是教学目标。

// Flow 是一条流的散列输入。
type Flow struct {
	Src, Dst uint32
	SPort    uint16
	DPort    uint16
	Proto    uint8 // 6=TCP 17=UDP
	Bytes    float64
	Label    string
}

// HashPolicy 对应 net.ipv4.fib_multipath_hash_policy。
type HashPolicy int

const (
	PolicyL3 HashPolicy = 0 // 三层：只看源目 IP
	PolicyL4 HashPolicy = 1 // 四层：IP + 端口 + 协议
)

// Mix 是 lookup3 的混合轮。
func mix(a, b, c uint32) (uint32, uint32, uint32) {
	a -= c
	a ^= c >> 4 & 0xffffffff
	c += b
	b -= a
	b ^= a << 6 & 0xffffffff
	a += c
	c -= b
	c ^= b >> 8 & 0xffffffff
	b += a
	a -= c
	a ^= c >> 16 & 0xffffffff
	c += b
	b -= a
	b ^= a << 19 & 0xffffffff
	a += c
	c -= b
	c ^= b >> 4 & 0xffffffff
	b += a
	return a, b, c
}

// Final 是 lookup3 的收尾轮。
func final(a, b, c uint32) uint32 {
	c ^= b
	c -= b>>13 | b<<19
	a ^= c
	a -= c<<8 | c>>24
	b ^= a
	b -= a>>13 | a<<19
	c ^= b
	c -= b>>12 | b<<20
	a ^= c
	a -= c<<16 | c>>16
	b ^= a
	b -= a>>5 | a<<27
	c ^= b
	c -= b>>3 | b<<29
	return c
}

// jhash2 复刻内核的 jhash2：按 32 位字吃输入。
func jhash2(k []uint32, init uint32) uint32 {
	n := uint32(len(k))
	a, b, c := 0xdeadbeef+n+init, 0xdeadbeef+n+init, 0xdeadbeef+n+init
	for len(k) > 3 {
		a += k[0]
		b += k[1]
		c += k[2]
		a, b, c = mix(a, b, c)
		k = k[3:]
	}
	switch len(k) {
	case 3:
		c += k[2]
		fallthrough
	case 2:
		b += k[1]
		fallthrough
	case 1:
		a += k[0]
		return final(a, b, c)
	}
	return c
}

// jhash2words 是两个字的快捷入口。
func jhash2words(a, b, init uint32) uint32 {
	return jhash2([]uint32{a, b}, init)
}

// FibMultipathHash 复刻内核的 fib_multipath_hash。
func FibMultipathHash(f Flow, p HashPolicy) uint32 {
	if p == PolicyL4 {
		h := jhash2([]uint32{f.Src, f.Dst}, 0)
		ports := uint32(f.DPort)<<16 | uint32(f.SPort)
		return jhash2([]uint32{ports, uint32(f.Proto)}, h)
	}
	return jhash2words(f.Src, f.Dst, 0)
}

// PickLink 是内核的选择方式：取模。权重相同时就是 hash % n。
func PickLink(h uint32, n int) int {
	if n <= 1 {
		return 0
	}
	return int(h % uint32(n))
}
