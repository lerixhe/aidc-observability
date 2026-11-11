#!/usr/bin/env python3
"""向指定网络接口直接注入真实 802.3x / 802.1Qbb 优先级流控（PFC）Pause 帧 (EtherType 0x8808)。

原理：
  使用 AF_PACKET 原始套接字向链路层注入标准 Pause 帧：
  - 目的 MAC: 01:80:C2:00:00:01 (标准流控组播地址)
  - 802.3x 标准控制帧类型: 0x8808 (Control Frames)
  - 操作码 (OpCode): 0x0101 (PFC / Priority-based Flow Control)
  - Class Enable Vector: 8 位的优先级掩码 (bit 3 置 1 代表 RoCE 常用队列 3)
  - Quanta: 对应优先级的暂停配额 (单位: 512 bit-times)
"""

import argparse
import socket
import struct
import sys
import time

DEST_MAC = b"\x01\x80\xc2\x00\x00\x01"  # IEEE 802.3x 标准组播地址
ETHERTYPE_PAUSE = 0x8808                 # 链路控制帧
OPCODE_PFC = 0x0101                      # PFC Opcode


def get_mac(iface: str) -> bytes:
    try:
        with open(f"/sys/class/net/{iface}/address", "r", encoding="utf-8") as f:
            mac_str = f.read().strip()
            return bytes.fromhex(mac_str.replace(":", ""))
    except Exception:
        # 回退使用缺省本端虚拟 MAC
        return b"\x02\x42\x0a\x00\x00\x01"


def build_pfc_frame(src_mac: bytes, priority: int = 3, quanta: int = 65535) -> bytes:
    """构建标准 PFC 帧载荷 (共 64 字节以满足以太网最小帧长要求)。"""
    eth_hdr = struct.pack("!6s6sH", DEST_MAC, src_mac, ETHERTYPE_PAUSE)
    
    # Class-enable vector (仅使能指定优先级，例如 priority 3 -> 0x0008)
    class_enable = (1 << priority) & 0xFFFF
    
    # 8 个优先级的 Quanta 数组
    quanta_list = [0] * 8
    quanta_list[priority] = quanta & 0xFFFF
    
    pfc_payload = struct.pack("!HH8H", OPCODE_PFC, class_enable, *quanta_list)
    
    # 填充到以太网帧最小 60 字节 (不含 4 字节 FCS，驱动会自动追加)
    pad_len = 60 - (len(eth_hdr) + len(pfc_payload))
    padding = b"\x00" * max(0, pad_len)
    
    return eth_hdr + pfc_payload + padding


def main():
    parser = argparse.ArgumentParser(description="AIDC PFC Frame Injector (0x8808)")
    parser.add_argument("-i", "--interface", default="eth1", help="目标网络接口 (默认: eth1)")
    parser.add_argument("-p", "--priority", type=int, default=3, help="暂停的队列优先级 (默认: 3)")
    parser.add_argument("-q", "--quanta", type=int, default=65535, help="Pause Quanta 配额 (默认: 65535)")
    parser.add_argument("-c", "--count", type=int, default=10, help="发送帧数 (默认: 10, 0 表示持续)")
    parser.add_argument("-r", "--rate", type=float, default=100.0, help="发包速率 frames/sec (默认: 100)")
    args = parser.parse_args()

    src_mac = get_mac(args.interface)
    frame = build_pfc_frame(src_mac, args.priority, args.quanta)

    try:
        sock = socket.socket(socket.AF_PACKET, socket.SOCK_RAW)
        sock.bind((args.interface, 0))
    except PermissionError:
        print(f"错误: 需要 root 权限或 CAP_NET_RAW 能力才能向 {args.interface} 注入原始以太网帧。", file=sys.stderr)
        sys.exit(1)
    except Exception as e:
        print(f"绑定接口 {args.interface} 失败: {e}", file=sys.stderr)
        sys.exit(1)

    print(f"[*] 开始向 {args.interface} 注入真实 PFC 优先流控帧 (0x8808):")
    print(f"    优先级: {args.priority} | Quanta: {args.quanta} | 目标帧数: {args.count} | 速率: {args.rate} fps")
    print("    可用命令抓包验证: tcpdump -i <interface> ether proto 0x8808 -xx -vv")

    sent = 0
    interval = 1.0 / args.rate if args.rate > 0 else 0.01

    try:
        while True:
            sock.send(frame)
            sent += 1
            if args.count > 0 and sent >= args.count:
                break
            time.sleep(interval)
    except KeyboardInterrupt:
        pass
    finally:
        sock.close()

    print(f"[+] 成功发送 {sent} 个真实 0x8808 PFC pause 帧。")


if __name__ == "__main__":
    main()
