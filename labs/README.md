# labs/ — 可运行的实验代码

> 状态：**v0.2 进行中**。L1 层已落地 4 个 lab，全部 `go run .` 即跑，无特权、无 Docker。

## 快速开始

```bash
cd labs/00-setup && make run    # 先体检：这台机器能跑哪些 lab
cd labs/02-pfc-sim && make compare   # 核心实验：PFC / ECN / DCQCN 场景对比
```

一次检查所有 lab：

```bash
./scripts/check-labs.sh    # vet + test + expected/ 自校验
```

## 设计约束

PFC / ECN 是硬件行为，纯软件仿真无法做到行为等价。因此 lab 分两层：

| 层 | 依赖 | 面向 | 特点 |
|---|---|---|---|
| **L1 数据面模拟** | 仅 Go / Python | 所有读者 | `go run .` 即跑，无特权、无 Docker |
| **L2 真拓扑仿真** | Docker + Linux（推荐 GitHub Codespaces） | 进阶读者 | containerlab + FRR，可注入真实故障 |

**每个 L1 lab 的 README 开头必须声明：「教学级模型，非硬件行为等价」**，并给出真机验证路径。

## lab 清单

| Lab | 层 | 对应课 | 内容 | 状态 |
|---|---|---|---|---|
| `00-setup` | L1 | 01 | 环境自检：工具版本、lab 盘点、冒烟检查 | ✅ v0.2 |
| `01-traffic-model` | L1 | 02 | IDC 小流 vs AllReduce 同步突发的到达率对照 | ✅ v0.2 |
| `02-pfc-sim` | L1 | 05-06 | PFC / ECN / DCQCN 参数化队列模型，输出时序与场景对比 | ✅ v0.2 |
| `04-nccl-trace` | L1 | 08 | 解析 NCCL 日志，还原流量矩阵、核算 busbw、定位 straggler | ✅ v0.2 |
| `03-fault-inject` | L2 | 07 | tc/netem + 伪造 pause 帧复现 HoL 与 incast | ⬜ v0.4 |
| `05-clos-topo` | L2 | 09 | containerlab 2 spine / 4 leaf，ECMP hash 极化演示 | ⬜ v0.4 |
| `06-gnmi-mock` | L1 | 11 | 可被 gNMI 客户端订阅的 mock server（OpenConfig 路径） | ⬜ v0.3 |
| `07-int-path` | L1 | 12 | INT 逐跳元数据的编码/解析玩具实现 | ⬜ v0.3 |
| `08-pipeline` | L1 | 14 | 秒级遥测 → 本地落盘（DuckDB）→ 查询保真度对比 | ⬜ v0.3 |
| `09-slo` | L1 | 15 | 由模拟指标流计算 SLI，对比三种 SLO 口径 | ⬜ v0.3 |
| `10-rca` | L1 | 16 | 吃模拟告警跑图算法做根因排序 | ⬜ v0.3 |
| `11-case-study` | L2 | 21 | 端到端排障剧本：从告警到定位到验证 | ⬜ v0.4 |

编号留空是给规划留位，不代表顺序执行。`00-setup` 建议先跑，其余互不依赖。

## 统一结构

```
labs/NN-<slug>/
├── README.md      # 目标 / 前置 / 命令 / 观察什么 / 思考题 / 模型边界
├── Makefile       # make run / make test / make expected / make clean
├── go.mod         # 每个 lab 独立 module，不共享依赖
├── *.go           # 模型与报告分离，便于读者改参数不改逻辑
└── expected/      # 预期输出样本，CI 用来做自校验
```

`expected/` 是这套东西能长期维护的关键：模型输出被锁进样本，改参数导致的行为变化
会在 diff 里显式出现，而不是悄悄发生过。

## 约定

- 每个 lab 独立可跑，不依赖其他 lab
- 零第三方依赖：只用 Go 标准库，避免读者被代理和版本问题挡住
- 输出优先纯文本 / CSV，图表自选工具绘制
- 固定随机种子的地方必须有 `expected/` 对照，保证可复现
- 不使用真实设备配置与内网地址；示例 IP 用 RFC 5737 文档地址段

## CI

`.github/workflows/labs.yml` 在 `labs/**` 变更时触发，执行 `scripts/check-labs.sh`：
每个 lab 过 `go vet` + `go test`，再核对 `expected/` 是否与当前输出一致。

改动模型后本地先跑一遍：

```bash
cd labs/<lab> && make expected && git diff
```

确认 diff 是有意的，再提交。

## Codespaces

仓库根有 `.devcontainer/`，Go 1.22 + Docker + Python 都装好。L2 拓扑实验（v0.4）
在 macOS 上需要特权和虚拟机，Codespaces 比本机省事。
