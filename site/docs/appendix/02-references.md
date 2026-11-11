---
title: 附录 B · 参考资料 —— 每条断言的出处
description: 全站 72 条外部引用的来源清单，按可信度分层组织，附核验结果与一条关于链接腐烂的经验。
---

这份清单有三个用途。核对某一课的机制断言出自哪里。判断一条说法值不值得信。找原始出处继续往下读。

它不负责把知识再过一遍。**引用相当于给坐标而不给结论。**

## 怎么用这份清单

这份清单的使用规则有三条。

第一条是先找一手来源。标准文本与官方文档能回答的，别去引二手的博客转述，因为转述最常见的错误是丢掉了适用条件。

第二条是引用要细到可验证的粒度。写"RFC 3168 定义了 ECN"没用，写"RFC 3168 第 5 节定义了 CE 标记位"才能被人核对。

第三条是厂商文档要给入口而不是深链。理由在末节，这一轮核验里已经有六个深链烂掉。

## 一、标准与规范

这一组全部属于一手来源，而且是其中可信度最高的一类文本。它们定义字段与行为，不定义实现的开销与默认值。

| 来源 | 覆盖什么 | 课 |
|---|---|---|
| [RFC 2474 — Definition of the Differentiated Services Field](https://www.rfc-editor.org/rfc/rfc2474.html) | IP 头的优先级字段与 DSCP，PFC 优先级映射的基础 | 04 |
| [RFC 3168 — The Addition of ECN to IP](https://www.rfc-editor.org/rfc/rfc3168.html) | ECN 标记位、CE 语义、端侧响应 | 02、04、06、15、21 |
| [RFC 5905 — NTPv4](https://www.rfc-editor.org/rfc/rfc5905.html) | 时钟同步协议，跨设备关联分析的前提 | 11、16 |
| [RFC 7665 — Service Function Chaining Architecture](https://www.rfc-editor.org/rfc/rfc7665.html) | NSH 封装背景，INT 常借这条路径承载 | 12 |
| [RFC 7938 — BGP in Large-Scale Data Centers](https://www.rfc-editor.org/rfc/rfc7938.html) | 大规模数据中心的路由设计与收敛要求 | 09 |
| [RFC 7950 — The YANG 1.1 Data Modeling Language](https://www.rfc-editor.org/rfc/rfc7950.html) | 数据建模语言，OpenConfig 的语法基础 | 11 |
| [RFC 8926 — Geneve](https://www.rfc-editor.org/rfc/rfc8926.html) | 通用网络虚拟化封装，遥测元数据的常见承载 | 00、12 |
| [RFC 9197 — IOAM Data Fields](https://www.rfc-editor.org/rfc/rfc9197.html) | 带内逐跳元数据的标准字段与封装语义 | 12、19、21 |
| [IEEE 802.1Qbb — Priority-based Flow Control](https://standards.ieee.org/ieee/802.1Qbb/4369/) | PFC 的帧格式、pause quanta 与逐跳反压语义（[另一入口](https://standards.ieee.org/ieee/802.1Qbb/4509/)） | 02、04、05、06、15、21 |
| [IEEE 802.1Qaz — Enhanced Transmission Selection](https://standards.ieee.org/ieee/802.1Qaz/5410/) | 优先级带宽分配，DCB 三件套之一 | 05 |
| [IEEE 802.1Q — VLAN 与优先级标记](https://standards.ieee.org/ieee/802.1Q/6844/) | PCP 字段与优先级到队列的映射 | 04 |
| [IEEE 802.3 — Ethernet](https://standards.ieee.org/ieee/802.3/7071/) | 以太网物理层与链路层，光层指标的定义依据 | 13 |
| [IEEE 1588 — PTP](https://standards.ieee.org/ieee/1588/4355/) | 精密时间协议，微秒级校时的标准 | 11 |
| [IBTA 官网](https://www.infinibandta.org/) | InfiniBand 联盟的规范与生态入口 | 20 |
| [IBTA 规范](https://www.infinibandta.org/ibta-specification/) | InfiniBand 架构与链路层，对比 RoCE 的基线 | 02、03、04、20 |
| [SNIA SFF 规范](https://www.snia.org/technology-communities/sff/specifications) | 光模块管理接口（SFF-8636 / SFF-8472），DOM 字段来源 | 13、21 |
| [IEEE 802.3 工作组](https://www.ieee802.org/3/) | 以太网标准的最新进展与草案入口 | 13 |
| [UALink Consortium](https://www.ualinkconsortium.org/) | 开放 scale-up 互联规范 | 10 |
| [Ultra Ethernet Consortium](https://ultraethernet.org/) | UEC 规范，面向 AI 与 HPC 的以太网增强 | 10 |

## 二、论文

这一组是论文，同样属于一手来源，但适用范围要看清。论文结论绑定它当时的硬件与流量模型。

| 来源 | 覆盖什么 | 课 |
|---|---|---|
| [Zhu et al., "Congestion Control for Large-Scale RDMA Deployments", SIGCOMM 2015](https://doi.org/10.1145/2785956.2787484) | DCQCN 原始论文，ECN 标记与降速闭环的出处 | 00、15 |
| [Kim et al., "In-band Network Telemetry via Programmable Dataplanes", SOSR 2015](https://dl.acm.org/doi/10.1145/2774993.2775007) | INT 的早期系统实现与开销测量 | 12 |

## 三、厂商与开源项目官方文档

这一组文档绑定具体实现，能回答这个产品支持什么，却答不出这个机制本身的原理。

| 来源 | 覆盖什么 | 课 |
|---|---|---|
| [NVIDIA NCCL 文档](https://docs.nvidia.com/deeplearning/nccl/user-guide/docs/) | 集合通信语义、拓扑探测、调优入口 | 00、01、10、22 |
| [NCCL — Collective Operations](https://docs.nvidia.com/deeplearning/nccl/user-guide/docs/usage/collectives.html) | AllReduce 等原语的等待关系 | 00、02、07、08 |
| [NCCL — Communicators](https://docs.nvidia.com/deeplearning/nccl/user-guide/docs/usage/communicators.html) | 通信子与拓扑亲和性 | 09 |
| [NCCL — Environment Variables](https://docs.nvidia.com/deeplearning/nccl/user-guide/docs/env.html) | 调试开关与算法选择 | 08 |
| [NCCL — Troubleshooting](https://docs.nvidia.com/deeplearning/nccl/user-guide/docs/troubleshooting.html) | 慢任务与超时的官方排查入口 | 19、21 |
| [NVIDIA DCGM](https://docs.nvidia.com/datacenter/dcgm/latest/) | GPU 侧指标采集，域内可观测的主要入口 | 10、19、21 |
| [NVIDIA XID Errors](https://docs.nvidia.com/deploy/xid-errors/) | XID 错误码含义与处置建议 | 01、10 |
| [NVIDIA NVLink](https://www.nvidia.com/en-us/data-center/nvlink/) | NVLink 与 NVSwitch 的代际带宽 | 10 |
| [NVIDIA DGX SuperPOD 文档](https://docs.nvidia.com/dgx-superpod/) | 整机组成与 fabric 设计 | 10 |
| [DGX SuperPOD H100 参考架构](https://docs.nvidia.com/dgx-superpod/reference-architecture-scalable-infrastructure-h100/latest/index.html) | 千卡级网络设计的公开参考 | 00、09 |
| [NVIDIA GPUDirect RDMA](https://docs.nvidia.com/cuda/gpudirect-rdma/) | 网卡直读显存的机制与限制 | 03 |
| [NVIDIA Networking 文档](https://docs.nvidia.com/networking/) | 交换机与网卡文档总入口 | 04、05、06 |
| [NVIDIA Spectrum-X](https://www.nvidia.com/en-us/networking/spectrumx/) | 以太网 AI Fabric 的自适应路由与拥塞控制 | 20 |
| [OpenConfig — public models](https://github.com/openconfig/public) | 厂商中立 YANG 模型集的全部内容 | 00、01、11、12、13、16、20、21、22 |
| [OpenConfig — release/models](https://github.com/openconfig/public/tree/master/release/models) | 发布版模型目录，含 interfaces 与 network-instances | 02 |
| [OpenConfig — `qos` model](https://github.com/openconfig/public/tree/master/release/models/qos) | 队列、PFC、ECN 计数器的字段定义 | 05、06、07、15 |
| [gNMI 规范](https://github.com/openconfig/gnmi) | RPC 定义、`Subscribe` 三模式与路径语义 | 11 |
| [openconfig/gnmic](https://github.com/openconfig/gnmic) | 开源 gNMI 采集器，订阅管理可读源码 | 11 |
| [gnmic 文档](https://gnmic.openconfig.net/) | 采集器的输出与缓冲配置 | 14 |
| [SONiC 文档](https://sonic-net.github.io/SONiC/) | 白盒交换机侧遥测的部署方式 | 04、11 |
| [Linux 内核 RDMA 子系统](https://docs.kernel.org/infiniband/index.html) | 内核侧 RDMA 栈的结构 | 03 |
| [linux-rdma/rdma-core](https://github.com/linux-rdma/rdma-core) | 用户态 RDMA 驱动与工具源码 | 03 |
| [P4 规范](https://p4.org/specifications/) | P4 语言与 INT 数据面规范 | 12 |
| [PyTorch — Distributed](https://pytorch.org/docs/stable/distributed.html) | 训练框架侧对集合通信的调用方式 | 08 |
| [Anthropic — Building Effective Agents](https://www.anthropic.com/research/building-effective-agents) | Agent 设计的模式与适用边界 | 18 |
| [Anthropic — Contextual Retrieval](https://www.anthropic.com/news/contextual-retrieval) | 检索增强的上下文构造方法 | 18 |

## 四、工程实践与方法论

这一组是二手材料，但其中多数已经算行业权威。它们给的是口径与流程，不是物理事实。

| 来源 | 覆盖什么 | 课 |
|---|---|---|
| [SRE Book — Managing Incidents](https://sre.google/sre-book/managing-incidents/) | 事故响应的角色分工与时间线 | 16、19 |
| [SRE Book — Postmortem Culture](https://sre.google/sre-book/postmortem-culture/) | 复盘文化，把耗时归因写进记录 | 18 |
| [SRE Book — 目录](https://sre.google/sre-book/table-of-contents/) | 全书章节索引，按需查阅 | 22 |
| [SRE Workbook — Implementing SLOs](https://sre.google/workbook/implementing-slos/) | SLI 定义式、错误预算与窗口选择 | 00、01、15 |
| [SRE Workbook — Monitoring](https://sre.google/workbook/monitoring/) | 监控分层与管道自身的监控 | 14 |
| [SRE Workbook — Postmortem Culture](https://sre.google/workbook/postmortem-culture/) | 复盘的制度化做法 | 19 |
| [PagerDuty Incident Response](https://response.pagerduty.com/) | 值班体系、事故分级与 MTTD/MTTR 度量 | 19 |
| [Release It!（第二版）](https://pragprog.com/titles/mnee2/release-it-second-edition/) | 稳定性模式：熔断、舱壁、超时与重试 | 16、18 |
| [NANOG 资源库](https://www.nanog.org/resources/) | 网络运维实践与案例分享 | 22 |

## 五、数据与存储组件

这一组是组件文档，它们回答这个系统怎么工作，不回答你的场景该选哪个。

| 来源 | 覆盖什么 | 课 |
|---|---|---|
| [Apache Kafka — Delivery Semantics](https://kafka.apache.org/documentation/#semantics) | `acks`、重试与至少一次投递 | 14 |
| [Apache Flink — Event Time and Watermarks](https://nightlies.apache.org/flink/flink-docs-stable/docs/concepts/time/) | 事件时间、水位线、迟到数据 | 14 |
| [ClickHouse 文档](https://clickhouse.com/docs) | 列存与聚合查询 | 17 |
| [ClickHouse — ReplacingMergeTree](https://clickhouse.com/docs/en/engines/table-engines/mergetree-family/replacingmergetree) | 幂等写入与去重的表引擎 | 14 |
| [Prometheus — Storage](https://prometheus.io/docs/prometheus/latest/storage/) | 保留期、压缩与分块 | 14、17 |
| [Prometheus — Histograms and summaries](https://prometheus.io/docs/practices/histograms/) | 分位数为何不能平均 | 01、15 |
| [Prometheus — Remote Write 规范](https://prometheus.io/docs/specs/remote_write_spec/) | 跨系统写入的协议约束 | 17 |
| [VictoriaMetrics 文档](https://docs.victoriametrics.com/) | 高基数时序存储的实现取舍 | 17 |
| [InfluxDB 文档](https://docs.influxdata.com/) | 时序库的另一条技术路线 | 17 |
| [Grafana — Query and transform](https://grafana.com/docs/grafana/latest/panels-visualizations/query-transform-data/) | 面板侧聚合与转换 | 17 |

## 六、开发环境与实验工具

| 来源 | 覆盖什么 | 课 |
|---|---|---|
| [Go — Downloads](https://go.dev/dl/) | L1 实验的唯一运行时依赖 | 01 |
| [Development Containers 规范](https://containers.dev/implementors/spec/) | `devcontainer.json` 的字段语义 | 01 |
| [GitHub Codespaces](https://docs.github.com/en/codespaces) | 云端开发环境与额度机制 | 01 |
| [Docker — Runtime privilege](https://docs.docker.com/engine/reference/run/#runtime-privilege-and-linux-capabilities) | 网络命名空间实验为何要特权 | 01 |
| [containerlab](https://containerlab.dev/) | L2 真拓扑实验的工具 | 01 |
| [本仓库 labs 说明](https://github.com/lerixhe/aidc-observability/blob/main/labs/README.md) | 实验清单、分层与约定 | 01 |

## 七、可信度分级

引用之前先判断来源层级，这相当于给来源定一个可信度档位。层级决定这句话能断言到什么程度。

| 层级 | 来源类型 | 能断言什么 | 不能断言什么 |
|---|---|---|---|
| 1 | 标准文本（RFC、IEEE、IBTA、SFF） | 字段语义、协议行为、互操作要求 | 具体实现的开销、默认值与性能 |
| 2 | 顶会论文 | 特定条件下实测到的行为与建模 | 别家设备的默认配置 |
| 3 | 厂商与开源官方文档 | 该实现支持什么、怎么配 | 别家实现的等价能力 |
| 4 | 工程实践与方法论（SRE、PagerDuty） | 口径定义、流程与度量方式 | 具体的阈值数字 |
| 5 | 技术博客、演讲、二手转述 | 线索与经验 | 断言，尤其在没给条件时 |

不够格的来源有三类。营销页把能力写成既成事实。二手转述会丢掉适用条件。没有日期的技术博客可能写于任何一年。

::: warning 厂商文档最容易踩的两个坑

第一个坑来自版本绑定这一件事。同一份文档的 latest 与特定版本页内容可能不同，引用时要写清版本。CUDA、DCGM、NCCL 的路径都带版本或 `latest`。

第二个坑来自能力与默认值混写。文档说支持 PFC 按优先级统计，这不等于默认开启。前者是能力，后者是配置，两者的工程含义差很远。
:::

## 八、核验记录

本清单涉及的 72 条外部链接，在 2026-11-10 用脚本核验过一轮。方法很朴素，也就是对每条发 GET 请求，跟随重定向，15 秒超时。

| 结果 | 条数 | 说明 |
|---|---|---|
| 200 正常 | 63 | 直接可达 |
| 403 拒绝脚本 | 9 | IEEE、ACM、DOI、NANOG 有反爬策略，浏览器可访问 |
| 404 已失效 | 6 | 第一轮发现，已全部修复并复验 |

六个 404 的修复方式值得记一笔。

| 原链接 | 问题 | 处理 |
|---|---|---|
| DGX SuperPOD 参考架构 | 路径改版 | 换到带版本段的 H100 参考架构地址 |
| Spectrum-X 文档 | 产品文档路径变更 | 换成产品页，不再指向深链 |
| P4 INT 规范 PDF | 静态文件被移走 | 换成 P4 规范总入口 |
| SOSR 2015 会议页 | 会议站点归档 | 换到 ACM DL 的论文 DOI |
| SRE Workbook 监控章节 | 章节路径调整 | 换成 `workbook/monitoring/` |
| PagerDuty 度量页 | 文档重组 | 换成文档总入口 |

## 九、一条坑

链接会腐烂，而且各个来源腐烂得不均匀。**RFC 编号是永久的，厂商文档的路径年年变。**

这一轮核验里六个失效链接全部来自厂商站、会议站与静态文件，RFC 一条没坏。这不是巧合，RFC 一条没坏也不是运气。标准组织把编号当契约，厂商把文档当产品页面。两个圈子的差别本质上就在这儿，前者拿稳定性当承诺，后者拿流量当目标。

所以引用规则按来源分两种。标准与论文给编号（RFC 7938、DOI 10.1145/2785956.2787484），编号本身就是坐标，而且永不变动。厂商文档给总入口加章节名，让读者自己去站内找当前路径。

把深链写进正文，就是把腐烂风险交给下一个维护者。**给入口，别给深链。**
