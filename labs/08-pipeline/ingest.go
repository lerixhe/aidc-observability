package main

// 采集器模拟的默认参数。这些数字对应课 14 的「五段链路五个丢点」。
const (
	DefaultBufferCap = 200 // 采集器本地缓冲容量（点）
	// 消费速率降档发生在劣化窗口起点，模拟「事故期间下游写入阻塞」
	SlowStartSec = DangerFromSec
	SlowEndSec   = DangerFromSec + 300
	NormalRate   = 1.0 // 稳态：消费能力刚好等于生产速率，没有余量
	SlowRate     = 0.1 // 降档期间：下游写入阻塞，消费能力掉一个数量级
	CatchUpRate  = 1.5 // 恢复后：扩容消费者追赶积压
)

// IngestResult 是一次采集器模拟的结果。
type IngestResult struct {
	Expected    int   // 期望点数
	Delivered   []Point
	Dropped     int
	FirstDropAt int // 首次丢弃的时刻，-1 表示没丢
	PeakBuffer  int
	BufferCap   int
}

// Completeness 返回整体完整率（0 到 1）。
func (r IngestResult) Completeness() float64 {
	if r.Expected == 0 {
		return 1
	}
	return float64(len(r.Delivered)) / float64(r.Expected)
}

// WindowCompleteness 返回指定窗口内的完整率。
func (r IngestResult) WindowCompleteness(fromSec, toSec int) float64 {
	expected := toSec - fromSec
	if expected <= 0 {
		return 1
	}
	got := 0
	for _, p := range r.Delivered {
		if p.TsSec >= fromSec && p.TsSec < toSec {
			got++
		}
	}
	return float64(got) / float64(expected)
}

// DroppedInWindow 返回某个窗口内被丢弃的点数。
func (r IngestResult) DroppedInWindow(fromSec, toSec int) int {
	if r.Expected == 0 {
		return 0
	}
	delivered := map[int]bool{}
	for _, p := range r.Delivered {
		delivered[p.TsSec] = true
	}
	n := 0
	for t := fromSec; t < toSec; t++ {
		if !delivered[t] {
			n++
		}
	}
	return n
}

// SimulateIngest 模拟「生产者 1 Hz、消费者速率会降档、缓冲有限」的采集器。
//
// 规则很直白：每 tick 先入队（满则丢），再按当前消费速率出队。
// 这就是一条有背压但没有兜底的管道：缓冲满了就开始静默丢弃。
func SimulateIngest(pts []Point, bufferCap int) IngestResult {
	if bufferCap <= 0 {
		bufferCap = DefaultBufferCap
	}
	res := IngestResult{Expected: len(pts), BufferCap: bufferCap, FirstDropAt: -1}

	buf := make([]Point, 0, bufferCap)
	carry := 0.0 // 消费速率的累积余量，避免浮点截断导致长期偏差

	for _, p := range pts {
		// 入队
		if len(buf) >= bufferCap {
			res.Dropped++
			if res.FirstDropAt < 0 {
				res.FirstDropAt = p.TsSec
			}
		} else {
			buf = append(buf, p)
			if len(buf) > res.PeakBuffer {
				res.PeakBuffer = len(buf)
			}
		}

		// 出队
		rate := NormalRate
		switch {
		case p.TsSec >= SlowStartSec && p.TsSec < SlowEndSec:
			rate = SlowRate
		case p.TsSec >= SlowEndSec && len(buf) > 0:
			rate = CatchUpRate // 积压没排空前一直追赶
		}
		carry += rate
		for carry >= 1 && len(buf) > 0 {
			res.Delivered = append(res.Delivered, buf[0])
			buf = buf[1:]
			carry--
		}
	}

	// 收尾：把缓冲里剩下的也算投递，否则会虚报丢弃
	res.Delivered = append(res.Delivered, buf...)
	return res
}
