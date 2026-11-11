# 发布稿 · v1.0

> 用途：站外发布的模板。三种长度，按投放渠道挑一段改。
> 发布时改的只有两处：站点链接、以及「你现在缺什么」那句个人化的开头。

---

## 一、长版（公众号 / 知乎 / V2EX，约 1200 字）

**《智算网络可观测指南》：从有损 IDC 到无损智算的机制、遥测与实验**

我从传统 IDC 转到智算网络时，遇到的第一个问题是：**原来那套监控面板，搬过去全部失效。**

不是指标不够多。是判据错了。IDC 里丢包是背景噪声，TCP 会自己重传。智算里丢包就是事故本身，
一次重传的代价可能是一个训练任务回退半小时。所以那边看带宽和丢包率，这边要看 PFC 时长、
ECN 标记率的斜率和队列水线的形状。

这套材料想解决的，就是「看什么、为什么、怎么看」这三个问题。

### 三个它和别处不一样的地方

**一、讲机制的资料很多，讲可观测这个横切面的很少。**

RoCE、PFC、ECN 的机制，厂商白皮书和 RFC 写得很清楚。但「一台交换机上几十个计数器，
哪个能说明正在劣化」这类判据，通常只存在于运维老手的脑子里。这门课把它写成了可复述的规则。

比如 PFC 风暴的指纹是「零丢包 + 多端口同期劣化 + 时延台阶式抬升」。
死锁的指纹是「超时 + 零丢包」这对矛盾组合。incast 的指纹是周期性，而且周期等于训练步的周期。
四类故障的告警长得像，区别全在遥测形态里。

**二、有能跑起来的实验，不只是代码片段。**

12 个 lab，10 个只需要 Go，`go run .` 直接出结果。剩下的 2 个要 containerlab 起真拓扑，
但默认模式也是零依赖 —— 先生成剧本和结论，`make deploy` 才是那一步需要 Docker。

实验覆盖：PFC 队列模型、NCCL 日志还原流量矩阵、gNMI 订阅保真度、INT 逐跳元数据、
遥测管道背压、三种 SLO 口径、告警收敛与图打分 RCA、ECMP 极化、四类故障的遥测指纹判读。

每个实验都有 `expected/` 目录，锁着上次人工确认过的输出。改了模型就有 diff，
要么你承认这是有意的行为变化，要么你发现了一个 bug。**先 diff 再提交，顺序不能反。**

**三、讲清了模型的边界。**

PFC 是硬件行为，容器里造不出真的 pause 帧。所以每个实验开头都有一句声明：教学级模型，
非硬件行为等价。同时给出真机验证路径 —— 交换机上该看哪几个计数器、怎么换算成占空比。

我最不想要的效果是，有人拿模型输出当真机读数引用。

### 内容清单

26 课，7 篇：认识这门课、两套世界观、不丢包的代价、流量与连线、
把现场变成数据、从数据到结论、路线与收口。外加 120 条术语表和 75 条参考资料，
参考资料按可信度分了七档。

课名一律两段式：主标题给判断，副标题给技术点，例如「把丢包换成堆积 —— PFC 的逐跳反压与它的两个代价」。
练习的答案折叠在 `<details>` 里，配评分维度，自己给自己打分。

### 怎么开始

```bash
git clone https://github.com/lerixhe/aidc-observability
cd aidc-observability/labs/00-setup && make run    # 这台机器能跑哪些实验
cd ../02-pfc-sim && make compare                   # 核心实验：PFC / ECN / DCQCN 对照
```

只需要 Go 1.22。想跑全部检查：`./scripts/check-labs.sh`。

### 一句实话

这套材料对着「刚接手一个智算集群、需要知道该盯哪些指标」这个场景写的。
如果你正在做这件事，它能省你几个月试错。如果你只是想了解 RoCE 机制，
厂商文档比我讲得全，这里多出来的是判据那一层。

- 站点：https://lerixhe.github.io/aidc-observability/
- 代码：https://github.com/lerixhe/aidc-observability
- 许可：内容 CC BY-SA 4.0，代码 MIT

v1.0 起课程结构冻结，只接受纠错类 PR。发现机制错误请带来源提 Issue。

---

## 二、短版（微博 / 朋友圈 / 群，约 200 字）

**《智算网络可观测指南》v1.0 发布。**

23 课讲清智算网络该看哪些指标：PFC 风暴、死锁、incast、队头阻塞四类故障的遥测指纹，
gNMI / INT 采集，指标体系与 RCA。配 12 个能跑的 lab，10 个只需要装个 Go。

关键的一点：四类故障的告警长得像，区别全在形态里。风暴是零丢包加时延台阶，
死锁是超时加零丢包，incast 是周期等于训练步的尖刺。这些判据课程里写成了可复述的规则，
lab 里写成了可执行的代码。

内容 CC BY-SA 4.0，代码 MIT。

https://lerixhe.github.io/aidc-observability/

---

## 三、英文短版（Hacker News / Reddit / r/networking）

**Show HN: AIDC Observability — telemetry and labs for AI training fabrics**

Observability for AI training networks (RoCEv2, lossless Ethernet) has plenty of
mechanism documentation and very few working reproductions. This project tries to
close that gap.

23 lessons (Chinese) plus 12 labs:

- PFC / ECN / DCQCN queue model with per-scenario comparisons
- NCCL log → traffic matrix, busbw accounting, straggler detection
- gNMI SAMPLE vs ON_CHANGE fidelity, INT hop metadata, pipeline backpressure
- Three SLI definitions of availability, error budgets, burn rate
- ECMP polarization: why 1:1 non-blocking still leaves one uplink idle
- A telemetry-fingerprint classifier for the four classic failure modes
  (PFC storm, deadlock, incast, head-of-line blocking)

Ten of the twelve labs need only Go 1.22 — no Docker, no privileges. The two
containerlab labs default to a zero-dependency mode that prints the injection plan
and the model output, with `make deploy` as the opt-in step.

Every lab locks its output in `expected/` and CI diffs it, so behavior changes
surface as diffs rather than as silent regressions.

Boundary is stated up front: containers cannot emit real PFC pause frames, so those
columns are modeled. Each README says which is which.

Content CC BY-SA 4.0, code MIT.

https://lerixhe.github.io/aidc-observability/
