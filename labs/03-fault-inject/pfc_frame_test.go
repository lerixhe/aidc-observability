package main

import (
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

// 这一组断言守的是「线上真实能解的帧格式」：
//
//	目的 MAC  01:80:c2:00:00:01   IEEE 802.3x 流控组播地址
//	EtherType 0x8808             链路层控制帧
//	OpCode    0x0101             PFC（802.1Qbb）
//	class-enable 位图            第 N 位对应优先级 N
//
// 判据来自标准而不是本项目。改脚本时如果这几条变了，说明帧不再是 PFC 了。

const (
	pfcDstMAC = "0180c2000001"
	pfcEther  = "8808"
	pfcOpcode = "0101"
)

// scriptFrame 调 python 脚本构造一帧，取 hex。python3 不在就跳过。
func scriptFrame(t *testing.T, prio, quanta int) string {
	t.Helper()
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("没有 python3，跳过帧格式集成校验")
	}
	code := `import sys; sys.path.insert(0,'scripts')
from inject_pfc_pause import build_pfc_frame
sys.stdout.write(build_pfc_frame(bytes.fromhex('020000000001'), ` +
		strconv.Itoa(prio) + `, ` + strconv.Itoa(quanta) + `).hex())`
	out, err := exec.Command("python3", "-c", code).Output()
	if err != nil {
		t.Fatalf("生成帧失败：%v", err)
	}
	return strings.TrimSpace(string(out))
}

func TestPFCFrameLayout(t *testing.T) {
	frame := scriptFrame(t, 3, 65535)

	if len(frame) != 2*60 {
		t.Fatalf("帧长 %d hex 字符，以太网最小 60 字节（不含 FCS）应为 120", len(frame))
	}
	if got := frame[0:12]; got != pfcDstMAC {
		t.Errorf("目的 MAC 是 %s，期望标准流控组播 %s", got, pfcDstMAC)
	}
	if got := frame[24:28]; got != pfcEther {
		t.Errorf("EtherType 是 %s，期望 %s", got, pfcEther)
	}
	if got := frame[28:32]; got != pfcOpcode {
		t.Errorf("OpCode 是 %s，期望 %s（PFC）", got, pfcOpcode)
	}
	// class-enable：优先级 3 → bit3 → 0x0008
	if got := frame[32:36]; got != "0008" {
		t.Errorf("class-enable 是 %s，期望 0008（只使能优先级 3）", got)
	}
	// 8 个 quanta，第 4 个是我们的 65535
	if got := frame[36+3*4 : 36+4*4]; got != "ffff" {
		t.Errorf("优先级 3 的 quanta 是 %s，期望 ffff", got)
	}
}

func TestPFCFramePriorityBitmask(t *testing.T) {
	for _, c := range []struct {
		prio int
		want string
	}{
		{0, "0001"},
		{1, "0002"},
		{3, "0008"},
		{7, "0080"},
	} {
		frame := scriptFrame(t, c.prio, 100)
		if got := frame[32:36]; got != c.want {
			t.Errorf("优先级 %d 的 class-enable 是 %s，期望 %s", c.prio, got, c.want)
		}
	}
}
