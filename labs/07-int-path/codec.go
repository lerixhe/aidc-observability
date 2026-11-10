package main

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// HopMeta 是一跳的 INT 元数据。字段与课程 12 的表格一一对应。
//
// 布局固定 24 字节，便于按固定步长滑窗解析：
//
//	0..3   switch id
//	4..5   ingress port
//	6..7   egress port
//	8..11  hop latency（纳秒）
//	12..15 出口队列占用（KB）
//	16..19 出口利用率（基点，1/10000）
//	20..23 保留（时间戳低位，本 lab 用不到）
const HopSize = 24

var (
	// ErrShortPayload 表示载荷长度不是 HopSize 的整数倍。
	ErrShortPayload = errors.New("载荷长度不是 24 字节的整数倍")
)

// HopMeta 是单跳元数据。
type HopMeta struct {
	SwitchID uint32
	Ingress  uint16
	Egress   uint16
	HopLatNs uint32
	QueueKB  uint32
	TxUtilBP uint32 // 基点：10000 表示 100%
}

// Name 给设备一个可读名字，便于报告对齐。
func (h HopMeta) Name() string {
	return fmt.Sprintf("sw-%04x", h.SwitchID)
}

// TxUtilPct 返回百分比形式的出口利用率。
func (h HopMeta) TxUtilPct() float64 { return float64(h.TxUtilBP) / 100 }

// Encode 把一串元数据编码成线格式。
func Encode(hops []HopMeta) []byte {
	buf := make([]byte, 0, len(hops)*HopSize)
	for _, h := range hops {
		var rec [HopSize]byte
		binary.BigEndian.PutUint32(rec[0:4], h.SwitchID)
		binary.BigEndian.PutUint16(rec[4:6], h.Ingress)
		binary.BigEndian.PutUint16(rec[6:8], h.Egress)
		binary.BigEndian.PutUint32(rec[8:12], h.HopLatNs)
		binary.BigEndian.PutUint32(rec[12:16], h.QueueKB)
		binary.BigEndian.PutUint32(rec[16:20], h.TxUtilBP)
		buf = append(buf, rec[:]...)
	}
	return buf
}

// Decode 按 24 字节固定步长滑窗解析，尾部残字节直接报错。
func Decode(payload []byte) ([]HopMeta, error) {
	if len(payload)%HopSize != 0 {
		return nil, fmt.Errorf("%w：长度 %d", ErrShortPayload, len(payload))
	}
	hops := make([]HopMeta, 0, len(payload)/HopSize)
	for off := 0; off+HopSize <= len(payload); off += HopSize {
		rec := payload[off : off+HopSize]
		hops = append(hops, HopMeta{
			SwitchID: binary.BigEndian.Uint32(rec[0:4]),
			Ingress:  binary.BigEndian.Uint16(rec[4:6]),
			Egress:   binary.BigEndian.Uint16(rec[6:8]),
			HopLatNs: binary.BigEndian.Uint32(rec[8:12]),
			QueueKB:  binary.BigEndian.Uint32(rec[12:16]),
			TxUtilBP: binary.BigEndian.Uint32(rec[16:20]),
		})
	}
	return hops, nil
}

// WireBytes 返回编码后的字节数，报告里用来算开销。
func WireBytes(n int) int { return n * HopSize }
