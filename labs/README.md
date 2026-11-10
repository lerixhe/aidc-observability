# labs/ — 可运行的实验代码

> 状态：**规划中**，v0.2 起逐步落地。当前目录为空。

## 设计约束

PFC / ECN 是硬件行为，纯软件仿真无法做到行为等价。因此 lab 分两层：

| 层 | 依赖 | 面向 | 特点 |
|---|---|---|---|
| **L1 数据面模拟** | 仅 Go / Python | 所有读者 | `go run .` 即跑，无特权、无 Docker |
| **L2 真拓扑仿真** | Docker + Linux（推荐 GitHub Codespaces） | 进阶读者 | containerlab + FRR，可注入真实故障 |

**每个 L1 lab 的 README 开头必须声明：「教学级模型，非硬件行为等价」**，并给出真机验证路径。

## 规划中的 lab

| Lab | 层 | 对应课 | 内容 |
|---|---|---|---|
| `00-setup` | L1 | 01 | 环境自检：Go 版本、Docker 可用性、可选 model 参数 |
| `01-traffic-model` | L1 | 02 | 对比 IDC 小流分布与 AllReduce 同步突发的流量模型 |
| `02-pfc-sim` | L1 | 05-06 | 参数化队列模型，输出 pause / ECN 标记时序（CSV + 图表） |
| `03-fault-inject` | L2 | 07 | tc/netem + 伪造 pause 帧复现 HoL 与 incast |
| `04-nccl-trace` | L1 | 08 | 解析 NCCL 日志 / 通信 trace，还原流量矩阵 |
| `05-clos-topo` | L2 | 09 | containerlab 2 spine / 4 leaf，ECMP hash 极化演示 |
| `06-gnmi-mock` | L1 | 11 | 可被 gNMI 客户端订阅的 mock server（OpenConfig 路径） |
| `07-int-path` | L1 | 12 | INT 逐跳元数据的编码/解析玩具实现 |
| `08-pipeline` | L1 | 14 | 秒级遥测 → 本地落盘（DuckDB）→ 查询保真度对比 |
| `09-slo` | L1 | 15 | 由模拟指标流计算 SLI，对比三种 SLO 口径 |
| `10-rca` | L1 | 16 | 吃模拟告警跑图算法做根因排序 |
| `11-case-study` | L2 | 21 | 端到端排障剧本：从告警到定位到验证 |

## 统一结构

```
labs/NN-<slug>/
├── README.md      # 目标 / 前置 / 命令 / 观察什么 / 思考题
├── Makefile       # make run / make clean
├── main.go        # 或 compose.yaml / *.py
└── expected/      # 预期输出样本，便于自校验
```

## 约定

- 每个 lab 独立可跑，不依赖其他 lab
- 输出优先用纯文本 / CSV，图表由读者自选工具绘制（避免引入重型依赖）
- 不使用真实设备配置与内网地址；示例 IP 用 RFC 5737 文档地址段
