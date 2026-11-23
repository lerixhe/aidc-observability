# 智算网络可观测指南 · AIDC Observability

> 从有损 IDC 到无损智算：机制、遥测与可运行的实验室。

[![Open in GitHub Codespaces](https://github.com/codespaces/badge.svg)](https://codespaces.new/lerixhe/aidc-observability)

**26 课 + 附录 A/B，12 个实验全部落地（10 个 L1 + 2 个 L2 真拓扑）。** 站点：<https://lerixhe.github.io/aidc-observability/> · [English README](README.en.md)

本仓是系列第二册。第一册 [IDC 网络可观测入门](https://github.com/lerixhe/idc-network-observability/)
讲有损网络里的采集、指标、判读与定位，写给从业务后端转进网络可观测的读者。按顺序先读那一册。

---

## 这门课解决什么问题

AI 训练网络（智算中心、AIDC、AI Fabric，也有人叫高性能网络）把网络可观测的基本假设全部翻转了：

| | 传统 IDC | 智算中心 |
|---|---|---|
| 流量模型 | 南北向小流、随机到达 | 东西向大象流、AllReduce 周期性同步突发 |
| 丢包语义 | 背景噪声，TCP 静默重传 | 事故本身，恢复代价极大 |
| 故障代价 | 单机 / 单批请求 | 小时 × 千卡，回退半小时 |
| 遥测粒度 | 分钟级 SNMP 轮询 | 秒级 / 亚秒级流式订阅 |

把 IDC 的监控面板原样搬过去必然失效。本课程逐层拆解这套新规则：RDMA / RoCEv2 的因果链、
PFC / ECN 的机制与失败模式、gNMI / INT 的采集口径、指标体系与 RCA 自动化。

## 差异化

讲 RoCE / PFC 机制的资料很多，从**可观测**这个横切视角切入的很少；能在容器里复现 PFC 风暴
与 incast 的动手实验室，更少。课件讲机制，`labs/` 给能跑起来复现的代码。

## 全书结构

| 部分 | 讲号 | 讲什么 |
|---|---|---|
| 起点 | 开篇 00–01 | 指标全绿任务却慢；把实验跑起来 |
| 一 · 范式差异 | 02–04 | 为什么 IDC 那套失效、RDMA 内核旁路、RoCEv2 与 DCB、自适应路由与乱序重组 |
| 二 · 无损机制 | 05–07 | PFC 逐跳反压、ECN 与 DCQCN、PFC 故障模式 |
| 三 · 流量与拓扑 | 08–10 | NCCL 集合通信、Clos 与 Rail、超节点与 Scale-up、NVLink 链路故障 |
| 四 · 遥测采集 | 11–14 | gNMI 订阅、INT 带内遥测、光模块故障、数据管道 |
| 五 · 分析与运维 | 15–19 | 指标口径与 SLO、AIOps 根因定位、时序存储选型、LLM 与运维、MTTR 拆解 |
| 六 · 选型与收口 | 20–22 | IB 与 RoCE、UEC 超以太网、端到端排障案例、能力自评 |
| 附录 | A / B | 术语表、参考资料 |

## 实验代码

`labs/` 下 12 个 lab，分两层：

| 层 | 依赖 | 数量 | 特点 |
|---|---|---|---|
| L1 数据面模拟 | 仅 Go | 10 | `go run .` 即跑，无特权、无 Docker |
| L2 真拓扑仿真 | Docker + containerlab | 2 | 起真拓扑、注入故障；默认模式仍然零依赖 |

L2 的 `03-fault-inject`、`05-clos-topo` 走双模式：`make run` 只出注入剧本与模型结论，
`make deploy` 才需要 containerlab。想跑真帧也行：

```bash
cd labs/03-fault-inject
make deploy          # 6 节点拓扑
make inject-pfc      # 向 veth 注入真实 0x8808 PFC pause 帧
docker exec clab-fault-inject-h3 tcpdump -i eth1 ether proto 0x8808 -c 3 -xx
```

PFC / ECN 是硬件行为，纯软件仿真做不到行为等价。每个 L1 lab 的 README 都在开头声明这条边界，
并给出真机验证路径。`expected/` 存了字节稳定的样本，CI 用 diff 守住输出，行为变化会在改动里显式出现。

## 目录结构

```
.
├── site/                  VitePress 文档站（正文在 site/docs，按部分分目录）
├── labs/                  Go / 容器化实验代码（12 个，每个独立 module）
└── scripts/               链接与锚点体检、Mermaid 语法检查、实验自校验
```

## 跑起来

```bash
# 站点
cd site && pnpm install && pnpm dev

# 第一个实验（只要 Go 1.22+）
cd labs/00-setup && make run

# 全部实验自校验
./scripts/check-labs.sh
```

macOS 上跑 L2 拓扑要建 veth、改网络命名空间，本机做不了。走 Codespaces：点上面的徽章，
环境里已备好 Go、Docker 与 containerlab，克隆完就能跑真拓扑。Docker 不可用时初始化脚本会跳过
containerlab 安装，L1 实验照常。

## 许可

- 文字与课程内容：[CC BY-SA 4.0](LICENSE)
- `labs/` 与 `scripts/` 代码：[MIT](LICENSE-CODE)

方案与结论欢迎各自验证后取用；正文内容请遵守 CC BY-SA 的署名与相同方式共享要求。

## 贡献

内容已完整发布，接受纠错类 Issue 与 PR。请标明具体讲次，并附可核实的依据：

- 机制类断言请给可验证来源（RFC / IEEE 标准 / 论文 / 厂商公开文档）
- 文中示例统一使用 RFC 5737 文档地址段，请勿提交真实设备配置、内网地址或未公开的架构细节
- 本地开发：`cd site && pnpm install && pnpm dev`
