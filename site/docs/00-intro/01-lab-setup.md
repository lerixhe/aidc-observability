---
title: 先把车打着 —— 五分钟搭起实验室
description: 三条环境路线任选一条，跑通环境自检与第一个队列模型实验。顺带说清 L1 与 L2 的分界，以及这套实验为什么能长期维护。
---

如果要用一句话来描述这一讲，我想应该是：先跑 L1 那十个实验，L2 缺什么都不影响你读下去。

这门课的实验分两层。L1 层只依赖 Go，`go run .` 直接出结果。L2 层要真拓扑，需要 Docker 和 Linux。

先把 L1 跑通，五分钟够。**多数人卡住的地方不是技术难点，是把两层混在一起准备了。**

## 学完这一讲，你应该能

- 从三条环境路线里选一条并跑通，其中本机路线只需要 Go
- 读懂 `labs/00-setup` 的体检报告：哪些实验能跑、缺什么工具、缺了会怎样
- 跑出第一个实验结果，并解释 `expected/` 目录为什么是这套实验能长期维护的关键
- 说清 L1 与 L2 的分界，以及为什么 L2 的实验排在后面

## 机制拆解

### 0x01 三层依赖，按需准备

| 层 | 依赖 | 覆盖的实验 | 缺了会怎样 |
|---|---|---|---|
| **只读层** | 无 | 全部课程的机制讲解 | 照样能读完，判读部分会缺手感 |
| **L1 计算层** | Go ≥ 1.22 | `00-setup`、`01-traffic-model`、`02-pfc-sim`、`04-nccl-trace`、`06-gnmi-mock`、`07-int-path`、`08-pipeline`、`09-slo`、`10-rca`、`11-case-study` | 十个实验全部跳过 |
| **L2 拓扑层** | Docker、Linux、containerlab | `03-fault-inject`、`05-clos-topo` | 只是注入与真散列跑不了，两个 lab 的剧本与模型仍可跑 |

分层的理由很实在。PFC 与 ECN 是硬件行为，软件仿真做不到行为等价。硬要一层做全，读者会在第一条 `docker` 命令处流失。

所以 L1 走"数据面模型"路线：用参数化的队列模型复现因果链和量级关系。L2 走"真拓扑"路线：容器里的交换机和真网卡，代价是特权与虚拟机。

L2 的两个 lab 都是双模式。默认模式只生成剧本与模型结论，不需要 Docker：`make run` 照常出结果，`make deploy` 才需要 containerlab。缺 Docker 时降级体验，不是全有或全无。

### 0x02 三条环境路线

**路线 A，本机直跑。** 装一个 Go，其余什么都不用。

```bash
go version              # 需要 1.22 或更高
cd labs/00-setup
make run
```

macOS、Linux、Windows 都可以。这条路线最省事，代价是没有 L2。

**路线 B，devcontainer。** 仓库自带 `.devcontainer/`，镜像基于 `go:1.22-bookworm`，装了 docker-in-docker 与 python3。

```bash
devcontainer up --workspace-folder .
```

容器起来后会自动跑一次环境自检，也就是 `postCreateCommand` 里那句 `go run .`。VS Code 里选 "Reopen in Container" 效果一样。

**路线 C，GitHub Codespaces。** 网页打开仓库，点 Code → Codespaces。代价是消耗免费额度，好处是 L2 实验在云端不用折腾虚拟机。

::: info 经验映射（IDC → AIDC）

这套环境设计对应你熟悉的"本地跑单测，CI 跑集成"。

L1 相当于本地单测：快、零依赖、随时跑。L2 相当于集成测试环境：慢、要外围系统、按需启动。把两者混成一条命令，等于要求每个开发者在本地装齐生产依赖。

对后端来说这个判断几乎不用想，但换成网络实验，很多人第一反应是"要装 Docker 才算完整"。
:::

### 0x03 体检报告怎么读

`labs/00-setup` 不模拟网络，它只回答一个问题：这台机器能跑哪些实验。输出三段。

```text
工具               版本           状态     说明
go               1.22.5       ✅
Docker           —            ❌      未安装，或不在 PATH 里
python3          3.12.3       ✅

lab 盘点（12 个已落地 / 共登记 12 个）
  目录                 层    需要工具                 对应课   可跑   说明
  00-setup           L1   go                   01      ✅    环境自检
  02-pfc-sim         L1   go                   05,06   ✅    PFC / ECN / DCQCN 参数化队列模型
  03-fault-inject    L2   go+docker+clab       07      ❌    缺 docker+containerlab · 四大故障的注入剧本与遥测指纹判读
```

三个字段值得多看一眼。

**可跑** 列里 `✅` 能跑，`❌` 缺工具，`—` 还没落地。**层** 列告诉你缺的是哪一层。**说明** 列在 `❌` 那行会写清缺哪个工具，不用自己猜。

加上 `-try` 会多跑一轮冒烟检查，对每个可跑的 L1 实验执行 `go vet` 与 `go test`。CI 用的就是这个开关，有失败则退出码为 1。

```bash
cd labs/00-setup
make try
```

### 0x04 跑出第一个结果

队列模型实验是本课程的核心实验，覆盖第 5、6 两讲。

```bash
cd labs/02-pfc-sim
make run          # 默认场景，打印时序与汇总
make compare      # PFC / ECN / DCQCN 三种场景对照
```

输出是纯文本与 CSV。CSV 落在 `out/` 目录，用任意工具画图都行。不给内置画图，是有意的：**图的形状要你自己画一遍才会记住。**

## 判读与实操

### 0x05 实验与课程的对应表

| 实验 | 层 | 对应课 | 跑它能验证什么 |
|---|---|---|---|
| `00-setup` | L1 | 01 | 这台机器能跑哪些实验 |
| `01-traffic-model` | L1 | 02 | 小流随机到达与同步突发的队列差异 |
| `02-pfc-sim` | L1 | 05、06 | 水线阈值、pause 反压、ECN 降速的因果 |
| `04-nccl-trace` | L1 | 08 | 从 NCCL 日志还原流量矩阵与 straggler |
| `06-gnmi-mock` | L1 | 11 | SAMPLE 与 ON_CHANGE 的保真度差异 |
| `07-int-path` | L1 | 12 | INT 逐跳元数据的编码，与采样率对微突发的捕获率 |
| `08-pipeline` | L1 | 14 | 四档落盘能不能查出峰值，背压丢点落在哪 |
| `09-slo` | L1 | 15 | 同一份数据三种口径的可用率差多少 |
| `10-rca` | L1 | 16 | 告警收敛与图打分定位根因 |
| `11-case-study` | L1 | 21 | 同样的数据，两条排障路径的时间账 |
| `03-fault-inject` | L2 | 07 | 四类故障的注入剧本与指纹判读 |
| `05-clos-topo` | L2 | 09 | 流数与 hash 口径怎么决定极化 |

### 0x06 为什么每个实验都有 expected/

`expected/` 里放的是上一次人工确认过的输出样本。CI 拿当前输出和它逐字比对。

这个设计解决一个长期的维护问题。模型参数一改，行为就变，而这种变化通常悄无声息。

有了对照样本，改动会在 diff 里显式出现。要么你承认这是有意的行为变化，并更新样本。要么你发现这是个 bug。

```bash
cd labs/02-pfc-sim
make expected     # 重新生成样本
git diff          # 先看差异，再决定提不提交
```

顺序不能反。**先 diff 再提交，先提交再 diff，等于把 bug 锁进标准答案。**

::: warning 自校验失败先看 diff，别急着重生成

CI 报 `expected/` 不一致时，正确的第一反应是看差异内容，判断模型改动是不是预期的。

直接 `make expected` 覆盖，会让"标准答案"跟着当前实现走。对照样本就退化成了快照，失去发现回归的能力。
:::

## 经验映射

| 你已有的做法 | 在这套实验里的位置 |
|---|---|
| 本地单测 + CI 集成 | L1 与 L2 分层 |
| 快照测试（snapshot test） | `expected/` 目录与 `make expected` |
| devcontainer / 开发环境镜像 | `.devcontainer/` 的 Go + Docker + Python 组合 |
| 特性开关控制实验环境 | `-duration`、`-mode` 等参数化开关 |
| 回归对比 | `scripts/check-labs.sh` 的逐实验核对 |

没有哪一项需要新工具。需要新接受的只有一件事：网络行为的验证，多数时候靠"对照两次输出的差异"，而不是断言某个值等于多少。

## 习题

**必答 Q1**：某同事在 macOS 上装了 Docker Desktop，`docker --version` 正常，跑到 L2 实验时报错。给出你的排查顺序。

::: details 参考答案与评分维度

第一步分清楚 `--version` 和 `info` 的区别。前者只读二进制，后者要连 daemon。macOS 上常见情况是二进制装了、Docker Desktop 或 OrbStack 没启动。

排查顺序如下。先跑 `docker info` 确认 daemon 活着。再确认容器能拿到特权与网络命名空间，`docker run --rm --privileged alpine ip link` 是个快速验证。接着看 containerlab 版本与内核要求。最后才怀疑实验本身。

如果这台机器是 Apple Silicon，还要多一层判断：镜像架构是否匹配。架构不对时，错误信息通常指向二进制无法执行，而不是网络配置。

**评分维度**：区分 `--version` 与 `info`（必需）。有分层排查顺序而不是直接改配置（必需）。提到架构问题（加分）。
:::

**必答 Q2**：`expected/` 对照样本和"每次改动都更新样本"这两种做法，差别在哪。给出一个反例。

::: details 参考答案与评分维度

差别在样本的地位。对照样本是外部契约，改了要解释。每次跟着实现更新，样本就退化成了当前行为的快照，只能反映"现在是什么"，不能反映"应该是什么"。

反例很具体。队列模型的阈值判断里有一个差一错误，某次改动让 pause 提前一个时间单位触发。如果样本跟着实现更新，diff 里只有一堆数字变化，没人会追问。如果先看 diff 再看改动意图，就会发现"时序整体左移了一个单位"这种不该出现的变化。

**评分维度**：说出样本是契约而非快照（必需）。给出具体反例且能指认 diff 里该看什么（必需）。提到评审流程的配合（加分）。
:::

**自测 Q3**：为什么 L1 实验要求零第三方依赖，只用 Go 标准库。

::: details 参考答案与评分维度

核心原因是流失率。第三方依赖意味着代理、版本、校验和、平台差异，每一样都能挡掉一批读者。

另一个原因是可审查性。零依赖的代码读者能一行行读完，模型的假设、参数的来源、公式的边界都在明处。引入绘图库或统计库之后，关键计算会藏进黑箱，教学价值反而下降。

**评分维度**：给出可复现性与流失率的理由（必需）。提到可审查性（加分）。
:::

**自测 Q4**：如果只在 L1 层做实验，你会漏掉哪一类认知。

::: details 参考答案与评分维度

会漏掉"真实的随机性"。L1 是有确定性的模型，参数一样，输出一样。真机上有抖动、有芯片排队实现的差异、有驱动与固件的怪癖。

也会漏掉"配置的肉身"。PFC 水线怎么设、ECN 门限怎么配、DCB 三件套怎么落到设备上，这些都是 L2 才能碰到的手感。

**评分维度**：指出模型与真机在随机性与实现细节上的差异（必需）。举出至少一个 L2 才能练到的动作（加分）。
:::

## 小结

这一讲可以压成一句话：**先跑 L1 那十个实验，L2 缺什么都不影响你读下去。**

依赖分三层。只读层什么都不用装，L1 只要一个 Go，L2 要 Docker 加 Linux 加 containerlab。环境三条路，本机直跑最省事，devcontainer 和 Codespaces 是给需要 L2 又不想折腾虚拟机的人。

体检报告看三个字段就够了：可跑、层、说明。冒烟检查加 `-try`，CI 用的就是这个开关。

实验能不能长期维护，全在 `expected/` 那份人工确认过的输出样本上。参数的改动一定会在 diff 里显形，顺序是先 diff 再提交，反了就等于把 bug 锁进标准答案。

::: tip 本课要点

**先跑 L1 那十个实验，L2 缺什么都不影响你读下去。**

- 三层依赖：只读层无依赖、**L1 只要 Go**、L2 要 Docker 与 Linux
- 三条路线：本机直跑、devcontainer、Codespaces。**先跑 L1，别一起准备**
- `labs/00-setup` 三段输出：工具体检、实验盘点、`-try` 冒烟检查
- `expected/` 是外部契约，**先 diff 再决定是否更新样本**
- 实验覆盖表在 `labs/README.md`，课与实验双向可查
- 一条坑：`docker --version` 通过不代表 daemon 可用，L2 排查从 `docker info` 开始
:::

留一个思考题：如果让你删掉一个 L1 实验以减少维护成本，你删哪个，怎么保证被判读能力的缺口不扩大。

## 参考资料

- [Go — Downloads](https://go.dev/dl/)（L1 实验唯一的运行时依赖，1.22 及以上）
- [Development Containers Specification](https://containers.dev/implementors/spec/)（`.devcontainer/devcontainer.json` 的字段语义）
- [GitHub Codespaces Documentation](https://docs.github.com/en/codespaces)（云端环境与额度机制）
- [containerlab Documentation](https://containerlab.dev/)（L2 拓扑实验的工具）
- [Docker — Privileged mode](https://docs.docker.com/engine/reference/run/#runtime-privilege-and-linux-capabilities)（为什么网络命名空间实验需要特权）
- 仓库内：[`labs/README.md`](https://github.com/lerixhe/aidc-observability/blob/main/labs/README.md)（实验清单与约定）
