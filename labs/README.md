# labs/ — 可运行的实验代码

> 状态：**v1.0**。12 个 lab 全部落地。L1 层 10 个，`go run .` 即跑，无特权、无 Docker。
> L2 层 2 个（真拓扑与故障注入）额外要 Docker + containerlab，但默认模式仍然零依赖。

## 快速开始

```bash
cd labs/00-setup && make run    # 先体检：这台机器能跑哪些 lab
cd labs/02-pfc-sim && make compare   # 核心实验：PFC / ECN / DCQCN 场景对比
cd labs/11-case-study && make run    # 端到端排障剧本：两条路径的 MTTR 对照
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
| **L2 真拓扑仿真** | Docker + Linux（推荐 GitHub Codespaces） | 进阶读者 | containerlab 起拓扑，可注入真实故障 |

L2 的两个 lab（`03-fault-inject`、`05-clos-topo`）是**双模式**：

| 模式 | 命令 | 依赖 | 得到什么 |
|---|---|---|---|
| 默认 | `make run` / `make plan` | 仅 Go | 注入剧本、遥测指纹样本、散列分布结论 |
| 真跑 | `make deploy` / `make collect` / `make inject` | Docker + containerlab | 真队列、真丢包、真计数器 |

这个设计的理由是：判据与剧本能在 CI 里跑、能被 diff 住，而真实注入只能在有特权的机器上跑。
两者分开，读者不会卡在第一条 `docker` 命令上。

**每个 L1 lab 的 README 开头必须声明：「教学级模型，非硬件行为等价」**，并给出真机验证路径。

## lab 清单

| Lab | 层 | 对应课 | 内容 | 状态 |
|---|---|---|---|---|
| `00-setup` | L1 | 01 | 环境自检：工具版本、lab 盘点、冒烟检查 | ✅ v0.2 |
| `01-traffic-model` | L1 | 02 | IDC 小流 vs AllReduce 同步突发的到达率对照 | ✅ v0.2 |
| `02-pfc-sim` | L1 | 05-06 | PFC / ECN / DCQCN 参数化队列模型，输出时序与场景对比 | ✅ v0.2 |
| `04-nccl-trace` | L1 | 08 | 解析 NCCL 日志，还原流量矩阵、核算 busbw、定位 straggler | ✅ v0.2 |
| `03-fault-inject` | L2 | 07 | 四大故障的注入剧本与遥测指纹判读器 | ✅ v0.4 |
| `05-clos-topo` | L2 | 09 | 2 spine / 4 leaf，流数与 hash 口径决定 ECMP 极化 | ✅ v0.4 |
| `06-gnmi-mock` | L1 | 11 | OpenConfig 路径树 + SAMPLE / ON_CHANGE 保真度对照 | ✅ v0.2 |
| `07-int-path` | L1 | 12 | INT 逐跳元数据的编码/解析，含采样捕获率对照 | ✅ v0.3 |
| `08-pipeline` | L1 | 14 | 秒级遥测 → 四档落盘 → 查询保真度对比，含背压丢点 | ✅ v0.3 |
| `09-slo` | L1 | 15 | 由模拟指标流算可用率，三种口径 + 错误预算 + 燃烧率 | ✅ v0.3 |
| `10-rca` | L1 | 16 | 告警收敛 + 图打分做根因定位，对比两种朴素基线 | ✅ v0.2 |
| `11-case-study` | L1 | 21 | 端到端排障剧本 + MTTR 五段时间账 | ✅ v0.4 |

编号留空是给规划留位，不代表顺序执行。`00-setup` 建议先跑，其余互不依赖。

## 统一结构

```
labs/NN-<slug>/
├── README.md      # 目标 / 前置 / 命令 / 观察什么 / 思考题 / 模型边界
├── Makefile       # make run / make test / make expected / make clean
├── go.mod         # 每个 lab 独立 module，不共享依赖
├── *.go           # 模型与报告分离，便于读者改参数不改逻辑
├── topo.clab.yml  # 仅 L2 有：containerlab 拓扑，接口与地址写在文件头
├── scripts/       # 仅 L2 有：真环境采集脚本
└── expected/      # 预期输出样本，CI 用来做自校验
```

`expected/` 是这套东西能长期维护的关键：模型输出被锁进样本，改参数导致的行为变化
会在 diff 里显式出现，而不是悄悄发生过。

## 约定

- 每个 lab 独立可跑，不依赖其他 lab
- 零第三方依赖：只用 Go 标准库，避免读者被代理和版本问题挡住
- 输出优先纯文本 / CSV，图表自选工具绘制
- 固定随机种子的地方必须有 `expected/` 对照，保证可复现
- **样本与产物分开**：`samples/` 里的输入样例（日志、CSV）必须入库，`.gitignore` 已开例外。
  `out/`、`data/` 是跑出来的东西，不入库
- 不使用真实设备配置与内网地址。单段示例用 RFC 5737 文档地址段（192.0.2.0/24）。
  多网段拓扑（如 05-clos-topo）用 RFC 1918 私网段，避开办公室常见的 10.0.0.0/24 与 192.168.0.0/16，
  免得读者误以为要往真实内网里配

## CI

`.github/workflows/labs.yml` 在 `labs/**` 变更时触发，执行 `scripts/check-labs.sh`：
每个 lab 过 `go vet` + `go test`，再核对 `expected/`（以及 `03-fault-inject` 的 `samples/`）
是否与当前输出一致。L2 的拓扑文件只做结构检查，不 deploy —— CI 里没有特权，也没有必要。

L2 的两个 lab 靠单测守住「拓扑与计划的一致性」：`topo_test.go` 解析 `topo.clab.yml` 的链路，
再核对注入命令里出现的每个接口名。这类错（计划写 `leaf2:eth2`、拓扑里其实是 `eth3`）
只有真跑才会暴露，而真跑要 Docker。所以提前在文本层对齐。

改动模型后本地先跑一遍：

```bash
cd labs/<lab> && make expected && git diff
```

确认 diff 是有意的，再提交。

## Codespaces

仓库根有 `.devcontainer/`，Go 1.22 + Docker + Python 都装好。L2 拓扑实验
在 macOS 上需要特权和虚拟机，Codespaces 比本机省事。

要跑 `make deploy`，容器里得有 `containerlab`：

```bash
bash -c "$(curl -sL https://get.containerlab.dev)"
```

macOS 本机想要 L2，走 Lima 或 OrbStack 起一台 Linux 虚拟机，别无捷径 ——
containerlab 需要建 veth、改网络命名空间，这些在 macOS 内核上做不了。
