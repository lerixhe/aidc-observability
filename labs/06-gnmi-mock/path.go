package main

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Kind 区分三类遥测数据，它们的采集策略完全不同（对应课程里的分级订阅）。
type Kind int

const (
	KindCounter Kind = iota // 累积计数，要两次采样相减才知道速率
	KindGauge               // 瞬时值，单点即可判读
	KindState               // 离散状态，适合 ON_CHANGE
)

func (k Kind) String() string {
	switch k {
	case KindCounter:
		return "counter"
	case KindGauge:
		return "gauge"
	case KindState:
		return "state"
	}
	return "unknown"
}

// Leaf 是 OpenConfig 风格路径树上的一片叶子。
type Leaf struct {
	Path  string
	Kind  Kind
	Value func(s *Signal, t time.Duration) any
}

// ifaces 是被模拟的接口。一台交换机上只留两个，够演示又不啰嗦。
var ifaces = []string{"Ethernet0", "Ethernet4"}

// queueName 是承载 RoCE 无损流量的优先级队列。
const queueName = "3"

// Leaves 展开整棵指标树。
//
// 路径按 OpenConfig 的惯例写：配置在 config/ 下，运行态在 state/ 下，
// 计数器在 state/counters/ 下。PFC 与 ECN 的按优先级细分在真实设备上
// 常落在厂商扩展里，这里沿用 openconfig-interfaces 的命名习惯。
func Leaves() []Leaf {
	var out []Leaf
	for _, ifc := range ifaces {
		base := fmt.Sprintf("/interfaces/interface[name=%s]", ifc)
		qos := fmt.Sprintf("/qos/interfaces/interface[id=%s]/output/queues/queue[name=%s]", ifc, queueName)

		out = append(out,
			Leaf{base + "/state/oper-status", KindState, func(s *Signal, t time.Duration) any {
				return s.At(t).OperStatus
			}},
			Leaf{base + "/state/counters/out-octets", KindCounter, func(s *Signal, t time.Duration) any {
				return s.At(t).OutOctets
			}},
			Leaf{base + "/state/counters/out-discards", KindCounter, func(s *Signal, t time.Duration) any {
				return s.At(t).Discards
			}},
			Leaf{base + "/state/counters/out-pfc-pause-frames", KindCounter, func(s *Signal, t time.Duration) any {
				return s.At(t).OutPFC
			}},
			Leaf{base + "/state/counters/ecn-marked-pkts", KindCounter, func(s *Signal, t time.Duration) any {
				return s.At(t).ECNMarked
			}},
			Leaf{qos + "/state/watermark-bytes", KindGauge, func(s *Signal, t time.Duration) any {
				return s.At(t).WatermarkB
			}},
			Leaf{qos + "/state/transmit-pkts", KindCounter, func(s *Signal, t time.Duration) any {
				return s.At(t).OutPFC // 教学模型里与 pause 同源，够用
			}},
		)
	}
	return out
}

// MatchLeaves 按 gNMI 路径语法筛选叶子。
//
// 支持三种写法，与 gNMI 的订阅语义对齐：
//
//	/interfaces/interface[name=Ethernet0]/state/counters  前缀匹配（取子树）
//	/interfaces/interface[name=*]/state/oper-status       单段通配
//	/interfaces/.../state/counters/out-octets             递归通配
func MatchLeaves(pattern string) ([]Leaf, error) {
	re, err := compilePath(pattern)
	if err != nil {
		return nil, err
	}
	var out []Leaf
	for _, lf := range Leaves() {
		if re.MatchString(lf.Path) {
			out = append(out, lf)
		}
	}
	return out, nil
}

// compilePath 把 gNMI 路径语法翻成正则。
func compilePath(pattern string) (*regexp.Regexp, error) {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return nil, fmt.Errorf("路径不能为空")
	}
	if !strings.HasPrefix(pattern, "/") {
		return nil, fmt.Errorf("路径必须以 / 开头，收到 %q", pattern)
	}
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(pattern); {
		switch {
		case strings.HasPrefix(pattern[i:], "..."):
			b.WriteString(".*")
			i += 3
		case pattern[i] == '*':
			b.WriteString("[^/]*")
			i++
		default:
			b.WriteString(regexp.QuoteMeta(string(pattern[i])))
			i++
		}
	}
	// 前缀匹配：请求写到子树时，返回该子树下的全部叶子
	b.WriteString("(/.*)?$")
	re, err := regexp.Compile(b.String())
	if err != nil {
		return nil, fmt.Errorf("路径 %q 无法解析：%w", pattern, err)
	}
	return re, nil
}
