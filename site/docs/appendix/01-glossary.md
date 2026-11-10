---
title: 附录 A · 术语表
description: 一百来个术语的中英对照与一句话释义，按主题分组，每条标出首次出现的课号，供回查与对齐口径用。
---

同一件事在这行里有四五个叫法，圈子不同带来叫法差异。PFC 有人叫基于优先级的流量控制，有人直接说 pause。队列水线有人叫 buffer occupancy，有人叫队列深度。

这份表既不定义理论也不推导机制，它负责把叫法对齐，并指出回哪一课看细节。

## 怎么用这份表

这份表的使用方式有三条约定。第一条是分组方式，这张表按主题分组，没有按字母排序。

你要查的多半是“这一类东西里那个词”，而不是“以 P 开头的词”。

第二条是每一条后面都标着对应课号，这里的释义只够让你认脸，机制细节要回那一课看。

第三条是英文全称只在首次出现时给出，后面一律用缩写并跟现场说法保持一致，免得你到工位上说不出口。

## 一、范式与协议

| 术语 | 一句话释义 | 课 |
|---|---|---|
| 智算中心（AIDC，AI Data Center） | 以大算力集群为主、东西向流量占主导的数据中心形态 | 02 |
| 东西向流量（East-West Traffic） | 机柜之间、服务器之间的横向流量，AI 训练的主体 | 02 |
| 南北向流量（North-South Traffic） | 进出数据中心的纵向流量，传统 IDC 的主体 | 02 |
| 同步训练（Synchronous Training） | 所有 worker 每步等齐再进下一步，任一环节慢就整体慢 | 02 |
| 集合通信（Collective Communication） | 多卡多机协同的通信原语，如 AllReduce、AllGather、ReduceScatter | 08 |
| AllReduce | 把各卡数据归约后分发回所有卡，训练里最高频的集合操作 | 08 |
| straggler（拖后腿者） | 比同伴慢的节点或链路，决定同步任务的完成时间 | 08 |
| 拓扑感知（Topology Awareness） | 通信库探测硬件连接关系，据此选择路径与顺序 | 10 |
| RDMA（Remote Direct Memory Access） | 绕过内核与 CPU，网卡直接读写对端内存 | 03 |
| 内核旁路（Kernel Bypass） | 数据路径不经过内核协议栈，省掉拷贝与中断开销 | 03 |
| 队列对（QP，Queue Pair） | RDMA 通信端点，发送队列与接收队列成对存在 | 03 |
| RoCEv2（RDMA over Converged Ethernet v2） | 把 RDMA 搬到以太网上，用 UDP/IP 封装，可路由 | 04 |
| InfiniBand（IB） | 原生无损的专有互联，自带集中管理与拥塞控制 | 04 |
| DCB（Data Center Bridging） | 以太网做无损的一组扩展，含 PFC、ETS、QCN | 04 |
| GPUDirect RDMA | 网卡直接读写显存，不经过主机内存 | 04 |
| MTU（Maximum Transmission Unit） | 单个报文的最大长度，内部网络常用 1500 或 9000 字节 | 12 |

## 二、无损机制

| 术语 | 一句话释义 | 课 |
|---|---|---|
| 无损网络（Lossless Network） | 靠逐跳反压与拥塞标记把丢包压到零的网络 | 05 |
| PFC（Priority-based Flow Control） | 按优先级逐跳反压，队列快满就发 pause 帧让上游停手 | 05 |
| pause 帧（PFC Pause Frame） | 携带暂停时长的控制帧，只作用于指定优先级 | 05 |
| pause quanta | PFC 的时间单位，1 quanta 等于 512 bit time，10 Gb/s 下约 51.2 µs | 15 |
| XOFF / XON 水线 | 触发暂停与解除暂停的队列阈值，两者之间有回差 | 05 |
| headroom（预留缓冲） | 为在途报文预留的空间，防止 pause 传播期间丢包 | 05 |
| 水线阈值（Watermark Threshold） | 队列占用到某个值就触发某个动作的界线 | 05 |
| PFC 风暴（PFC Storm） | 反压沿拓扑连锁扩散，大范围链路被反复暂停 | 07 |
| PFC 死锁（Deadlock） | 互相等待的循环反压，谁也无法前进 | 07 |
| 队头阻塞（HoL，Head-of-Line Blocking） | 一个出口慢，同队列里其他流量一起被挡住 | 07 |
| incast（多打一） | 多个上游同时发往同一个出口，出口瞬时过载 | 07 |
| ECN（Explicit Congestion Notification） | 交换机在队列将满时给报文打标记，端侧看到就降速 | 06 |
| CE 标记（Congestion Encountered） | ECN 的拥塞标记位，打在 IP 头里 | 06 |
| DCQCN | RoCE 上的端网协同拥塞控制，收到 CNP 就降速、按定时器恢复 | 06 |
| CNP（Congestion Notification Packet） | 接收端回给发送端的拥塞通知报文 | 06 |
| ECN 标记率 | 被标记报文数与该优先级发出报文数之比，分母必须写死 | 15 |

## 三、流量与拓扑

| 术语 | 一句话释义 | 课 |
|---|---|---|
| Clos | 多级交换的全互联结构，靠等价多路径分摊流量 | 09 |
| Spine-Leaf | Clos 的两层常见形态，Leaf 接服务器、Spine 做互联 | 09 |
| 超订比（Oversubscription） | 下行总带宽与上行总带宽之比 | 09 |
| Rail（轨） | 按网卡序号切分的网络平面，同序号网卡连同一 rail | 09 |
| rail-optimized（轨优化） | 集合通信顺着 rail 走，避免跨 rail 跳转 | 09 |
| ECMP（Equal-Cost Multi-Path） | 等价多路径，按哈希把流分摊到多条路径 | 09 |
| 哈希极化（Hash Polarization） | 哈希不均导致部分链路拥塞、其余空闲 | 09 |
| 超节点（Superpod） | 几十到几百 GPU 装在一柜或相邻几柜、域内全互联的整机形态 | 10 |
| scale-up（纵向扩展） | 域内 GPU 之间用私有高速互联通信，内存语义 | 10 |
| scale-out（横向扩展） | 跨节点用网卡与交换机通信，报文语义 | 10 |
| NVLink | GPU 之间的高速点对点链路，带宽按链路条数叠加 | 10 |
| NVSwitch | NVLink 的全互联交换芯片与托盘 | 10 |
| 域（Domain） | 由 NVLink 全互联的 GPU 集合，域内通信不经过网卡 | 10 |
| NCCL | GPU 集合通信库，负责探测拓扑、构建 ring 与 tree | 08 |
| Ring / Tree | 两种集合通信算法形态，前者按环走、后者按树归约 | 08 |
| busbw（Bus Bandwidth） | NCCL 报告的有效带宽，由算法带宽按 2(n-1)/n 换算得到 | 08 |

## 四、遥测数据面

| 术语 | 一句话释义 | 课 |
|---|---|---|
| SNMP | 老式设备管理协议，拉模式、分钟级、对象平铺 | 11 |
| MIB / OID | SNMP 的数据模型与对象编号 | 11 |
| gNMI（gRPC Network Management Interface） | 基于 gRPC 的设备接口，支持流式订阅 | 11 |
| OpenConfig | 厂商中立的 YANG 模型集，统一设备数据路径 | 11 |
| YANG | 数据建模语言，定义设备的配置与状态树 | 11 |
| SAMPLE / ON_CHANGE / POLL | 订阅的三种模式：周期推、变化推、按需拉 | 11 |
| INT（In-band Network Telemetry） | 报文经过每跳时被写入本地元数据的遥测方式 | 12 |
| IOAM（In-situ OAM） | IETF 标准化的带内运维数据字段，INT 的标准化路线 | 12 |
| telemetry report | 交换机把元数据打包后单独上报的报文形态 | 12 |
| postcard | 每跳各自上报一行本地观察的轻量方案 | 12 |
| hop latency | 报文在该跳停留的时长，含排队与转发 | 12 |
| DOM（Digital Optical Monitoring） | 光模块自报的收发光功率、温度、偏置电流 | 13 |
| FEC（Forward Error Correction） | 链路层前向纠错，代价是延迟与少量带宽 | 13 |
| CRC | 链路层校验，错一个 bit 就整帧丢弃 | 13 |
| pre-FEC BER | 前向纠错之前的误码率，劣化早期最灵敏的信号 | 13 |
| lane | 光模块里的并行通道，单 lane 劣化只掉一份带宽 | 13 |

## 五、数据管道

| 术语 | 一句话释义 | 课 |
|---|---|---|
| 时间序列（Time Series） | 以时间戳与标签为索引的数值序列 | 14 |
| 基数（Cardinality） | 标签组合的数量，直接决定存储与查询成本 | 14 |
| 降采样（Downsampling） | 把细粒度数据聚合成粗粒度，本质是丢弃 | 14 |
| 峰值保留（Peak Retention） | 降采样时保留 max 与分位数，不只留均值 | 14 |
| 事件时间（Event Time） | 事件真实发生的时刻，报文自带 | 14 |
| 处理时间（Processing Time） | 数据被流处理器处理的时刻，重放会变 | 14 |
| 水位线（Watermark） | 系统对「数据到齐」的进度声明，窗口按它触发 | 14 |
| 迟到数据（Late Data） | 水位线之后才到达的事件，靠 allowed lateness 兜住 | 14 |
| 背压（Backpressure） | 下游处理不过来时向上游传导的减速信号 | 14 |
| 幂等写入（Idempotent Write） | 同一批数据重复写入不产生副作用 | 14 |
| 序列完整率（Sequence Completeness） | 实际点数与期望点数之比，按序列统计才有诊断价值 | 14 |
| TSDB（Time Series Database） | 专门存时间序列的数据库 | 17 |
| 列存（Columnar Store） | 按列组织的存储，适合时序聚合与压缩 | 17 |

## 六、指标与运维

| 术语 | 一句话释义 | 课 |
|---|---|---|
| SLI（Service Level Indicator） | 好事件占有效事件的比例，一个可测量的数 | 15 |
| SLO（Service Level Objective） | SLI 的目标值，必须带统计窗口 | 15 |
| SLA（Service Level Agreement） | 对外的承诺，通常带赔付条款 | 15 |
| 错误预算（Error Budget） | 1 减 SLO 之后允许的失败额度 | 15 |
| 燃烧率（Burn Rate） | 预算消耗速度，1 表示按 SLO 边界匀速消耗 | 15 |
| P50 / P95 / P99 | 分位数，比均值更能反映尾延迟 | 15 |
| 直方图（Histogram） | 分桶计数结构，跨实例聚合后可重算分位数 | 15 |
| 告警收敛（Alert Correlation） | 把大量派生告警压成少数 incident | 16 |
| RCA（Root Cause Analysis） | 定位故障源头的分析过程 | 16 |
| AIOps | 用算法做告警、关联与定位的运维方向 | 16 |
| MTTD / MTTI / MTTR / MTTV | 发现、定位、恢复、验证四段耗时，四者之和才是用户感知 | 19 |
| runbook | 标准化的处置步骤清单，含判据与回滚 | 19 |
| 假恢复（False Recovery） | 告警消失但性能没回基线 | 19 |
| 熔断（Circuit Breaking） | 连续失败后自动停下等人介入 | 19 |

## 七、硬件与主机

| 术语 | 一句话释义 | 课 |
|---|---|---|
| HBM（High Bandwidth Memory） | GPU 上的高带宽显存 | 08 |
| DCGM（Data Center GPU Manager） | GPU 指标采集组件，域内可观测的主要入口 | 10 |
| NVML | NVIDIA 管理库，DCGM 的底层接口 | 10 |
| XID | GPU 硬件错误码，靠查表区分可恢复与不可恢复 | 10 |
| ECC | 显存纠错，报错计数上升说明显存劣化 | 10 |
| 降频（Throttling） | 因温度或功耗触发频率下调 | 10 |
| NUMA | 非一致性内存访问，跨 socket 访存更慢 | 10 |
| 带外管理（Out-of-Band Management） | 不依赖业务网络的独立管理通道 | 10 |
| NTP / PTP | 网络时间协议与精密时间协议，后者精度更高 | 11 |

## 八、最容易混淆的六对

| 一对术语 | 区别一句话 |
|---|---|
| **PFC 与 ECN** | PFC 让上游停手，ECN 让上游降速。前者是最后防线，后者是常规手段 |
| **scale-up 与 scale-out** | 域内是内存语义、没有队列可溢出。域间是报文语义、要 PFC 与 ECN |
| **INT 与 gNMI** | INT 回答"这条流在哪一跳卡住"，gNMI 回答"哪个端口在冒烟" |
| **时间口径与事件口径** | 分母是分钟还是请求。夜间低流量事故在时间口径上重、事件口径上轻 |
| **降采样与压缩** | 压缩不丢信息，降采样丢。留均值等于把尖峰丢干净 |
| **MTTD 与 MTTI** | 前者是多久才发现，后者是发现之后多久定到位 |

::: tip 查词的三条捷径

只记得中文叫法时就直接搜“队列水线”，第二组与第五组都能对上。只记得缩写时就直接搜缩写本身，因为英文全称就写在同一行。

想确认两个词是不是同一件事就先看第八组，六对最容易混的都在那儿。
:::

## 九、术语为什么会漂移

同一个机制在三个圈子里有三种叫法，每一方都只说了一半。芯片厂商按寄存器命名，运维按现象命名，论文按机制命名。

**这种漂移有一个统一成因：每一方只描述自己看得见的那一段。** 芯片侧看得见队列与阈值两个量，于是把它叫作水线触发。运维侧看得见的是端口被暂停，于是把它叫作 pause 高。论文关心端到端的行为，于是把它叫作逐跳反压。

三个名字其实指向同一条因果链上的三个位置，这也是这份表采用分组方式的由来。分组逻辑跟查索引是同一个道理，也就是按你会怎么找东西，而不是按字母怎么排列。同一组里的词，大概率是同一条因果链上的邻居。

这里的判断翻译一下就很直白。查术语相当于看一张地名对照表，它的价值不在于收录了多少地名，而在于让你知道这两条路其实是同一条。

还有一条。最后再补一条使用上的建议，遇到一个陌生词时先做两个判断，一是它描述的到底是状态还是动作，二是它站在发送端还是接收端。

这两个问句能解掉大半歧义，比如 pause 描述的是一个动作，队列水线描述的是一个状态，两者不在同一个层面。ECN 标记由交换机做，DCQCN 降速由网卡做。前者发生在路径上，后者发生在端点上。

顺序别弄反。这类判断比背定义有用得多。术语表本质上不负责让你记住这些词，它负责让你在具体场景里快速对得上号。

## 参考资料

术语的释义以标准与官方文档为准，本表只做浓缩，需要细节时请回原始文档核对。

- [IEEE 802.1Qbb — Priority-based Flow Control](https://standards.ieee.org/ieee/802.1Qbb/4369/)（PFC 与 pause quanta 的定义）
- [RFC 3168 — The Addition of ECN to IP](https://www.rfc-editor.org/rfc/rfc3168.html)（ECN 标记位与端侧语义）
- [RFC 8926 — Geneve](https://www.rfc-editor.org/rfc/rfc8926.html) 与 [RFC 9197 — IOAM Data Fields](https://www.rfc-editor.org/rfc/rfc9197.html)（INT 与带内遥测的标准字段）
- [RFC 7950 — The YANG 1.1 Data Modeling Language](https://www.rfc-editor.org/rfc/rfc7950.html)（YANG 与数据模型）
- [OpenConfig — public models](https://github.com/openconfig/public)（OpenConfig 路径命名）
- [NVIDIA NCCL Documentation](https://docs.nvidia.com/deeplearning/nccl/user-guide/docs/)（集合通信与 busbw 的换算口径）
- [NVIDIA XID Errors](https://docs.nvidia.com/deploy/xid-errors/)（XID 错误码含义）
- [Google SRE Workbook — Implementing SLOs](https://sre.google/workbook/implementing-slos/)（SLI、SLO、错误预算、燃烧率）
- [Prometheus — Histograms and summaries](https://prometheus.io/docs/practices/histograms/)（直方图与分位数聚合）
