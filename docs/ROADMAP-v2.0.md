# AIDC Observability 架构与工程升级路线（ROADMAP-v2.0）

> 本文件是打破 v1.0 冻结期后的工程与演进蓝图。
> 目标：将本项目从「教学大纲型知识库」升级为「可运行、可交付、覆盖下一代架构的智算网络工程工具箱」。

---

## 阶段规划概览

```text
阶段一：内容补全（扩充 3 讲，覆盖技术代差）
    │
阶段二：遥测落盘与面板闭环（Lab 输出对接真实观测栈）
    │
阶段三：仿真环境降门槛与 L2 真实化（eBPF + Codespaces 预构建）
    │
阶段四：工具链与工程基建硬化（容错、CI 提速、多格式导出）
```

---

## 阶段一：补齐前沿技术盲区（课时 23 → 26）

当前课程停留在「静态哈希 ECMP + 标准 RoCEv2 + Scale-Out 交换网」。在 20,000+ 卡规模与超大模型训练场景下，该组合已现瓶颈。需追加 3 个核心课时：

### 1. 新增课时规范

| 课号与路径 | 篇归属 | 核心命题 | 交付内容与指标 |
|---|---|---|---|
| `site/docs/03-traffic-topo/10b-scale-up-nvlink.md` | 第三篇 · 流量与连线 | 节点内盲区：NVLink / NVSwitch 与 PCIe 降速如何拖垮整网 | 解析 `nvidia-smi nvlink` 错误计数、XID 故障指纹表，给出 NVLink 局部掉速与网络拥塞的因果判读法。 |
| `site/docs/01-paradigm/04b-adaptive-routing.md` | 第一篇 · 两套世界观 | 散列救不了的，路由来救：自适应路由（AR）、Packet Spraying 与网卡乱序重组 | 拆解 Broadcom DLB、NVIDIA AR 的选路与保序机制，给接收端乱序报文率（Out-of-Order Packets）的遥测指标。 |
| `site/docs/06-selection/20b-uec-next-gen.md` | 第六篇 · 路线与收口 | 走向超以太网：UEC 规范如何重写传输层与链路层 | 对比 RoCEv2 与 UEC 架构，解析基于 Credit 的反压机制与直通无损协议设计。 |

### 2. 课程配套更新

- `site/docs/03-traffic-topo/09-topology.md`：增加 1 节「从 Clos 静态选路到网包喷洒（Packet Spraying）的代价」。
- `site/docs/appendix/01-glossary.md`：补入 DLB、UEC、Packet Spraying、OOO 等 15 条核心术语。

---

## 阶段二：打通可观测闭环（Lab 输出对接真实看板）

当前 12 个实验全为 CLI 纯文本输出，缺乏生产可观测的直观感知。

### 1. 引入本地轻量可视化栈

- **涉及目录**：`labs/06-gnmi-mock/`、`labs/08-pipeline/`、`labs/docker-compose.telemetry.yml`
- **改造内容**：
  1. 在 `labs/06-gnmi-mock` 中内置 Prometheus Exporter 端点（`:9100/metrics`），将 mock 的端口速率、pause 计数、ECN 标记实时暴露为标准指标。
  2. 新增包含 VictoriaMetrics/Prometheus + Grafana 的最小容器编排文件，启动内存 ≤ 150MB。
  3. 提供 2 份开箱即用的 JSON 仪表盘模板：
     - `aidc-fabric-overview.json`：展示端口利用率、ECMP 极化热力图、逐跳 pause 速率。
     - `pfc-storm-trace.json`：重现 PFC 风暴扩散与队列水线堆积时序图。

### 2. 拓扑图与因果链路可视化输出

- **涉及文件**：`labs/05-clos-topo/report.go`、`labs/10-rca/report.go`
- **改造内容**：
  1. 为 `05-clos-topo` 增加 `--format mermaid` 开关，直接生成链路负载着色的 Mermaid 图，方便粘贴到文档或直接在 GitHub 渲染。
  2. 为 `10-rca` 增加 `--format dot` 输出，导出带异常置信度标色的故障传播图。

---

## 阶段三：降低仿真门槛与提升真实度

### 1. Codespaces 一键即开即跑

- **涉及文件**：`.devcontainer/devcontainer.json`、`README.md`
- **改造内容**：
  1. 预装 `containerlab`（v0.56+）与 Open vSwitch 依赖进 devcontainer 镜像。
  2. `README.md` 顶端配置 GitHub 一键部署徽章，免除 macOS 用户配置 Linux 虚拟机的摩擦。

### 2. L2 真拓扑引入 XDP / eBPF 软仿真（提升 PFC 真度）

- **现状局限**：容器虚拟接口默认无法生成硬件 PFC pause 帧，目前 `03-fault-inject` 依赖数学模型补足。
- **改动动作**：
  1. 编写 80 行以内的最小 XDP 程序，挂载于容器的 veth 接口。
  2. 当接收队列水线触顶时，直接在驱动层拦截后续报文，向入方向注入真实 `0x8808` 优先流控帧。
  3. 读者可用真实 `tcpdump -i <veth> ether proto 0x8808` 抓到 pause 帧，打破纯模型推演的隔阂。

---

## 阶段四：脚本鲁棒性硬化与体检升级

加固现有工具链，消除路径传参异常。

### 1. 脚本入参健壮化

- **涉及文件**：`scripts/style-check.py`、`scripts/long-sents.py`
- **改造内容**：
  1. 统一入参逻辑：既支持单个文件，也支持传入目录（如 `site/docs`）自动递归提取全部 `.md`。
  2. 修复 `scripts/check-sensitive.sh` 的环境依赖声明，防止使用不同 shell 时报错。

### 2. CI 体检项扩充

- **涉及文件**：`.github/workflows/docs.yml`
- **改造内容**：
  1. 引入 VitePress 坏链全站自动爬虫检查（不仅验证相对路径文件是否存在，同时验证 HTML 锚点 `#id` 存活性）。
  2. 将 Mermaid 渲染语法校验纳入 Docs CI，防止格式损坏。

---

## 排期与交付检查单

| 里程碑 | 工期估算 | 交付产物 | 验收命令 |
|---|---|---|---|
| **M1: 脚本加固与工具修复** | 1 天 | 修复入参报错，统一递归扫描支持 | `python3 scripts/style-check.py site/docs` 退出码 0 |
| **M2: 监控面板与 Exporter 闭环** | 3 天 | `06-gnmi-mock` Exporter、Grafana 仪表板模板 | `docker compose -f labs/docker-compose.telemetry.yml up -d` 见实时曲线 |
| **M3: 3 篇核心进阶课落地** | 4 天 | 10b / 04b / 20b 课程正文与外部参考文献 | `python3 scripts/check-stats.py`（数字更新至 26 课） |
| **M4: Codespaces 与 XDP 真帧仿真** | 3 天 | Devcontainer 预装环境，XDP pause 注入脚本 | `make deploy && make inject-pfc` 在抓包中见真实 pause 报文 |
