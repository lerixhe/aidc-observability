# 06-gnmi-mock · 订阅策略保真度

> **教学级模型，非硬件行为等价。** 指标信号是解析式生成的，传输层是 HTTP+SSE。
> 语义（路径语法、订阅模式、计数器口径）按 gNMI 对齐，协议本身不是 gRPC。

## 1. 这个 lab 做什么

模拟一台交换机，对外提供 OpenConfig 风格的指标树，然后拿同一份信号试不同的订阅策略：

- `SAMPLE` 100 ms / 1 s / 5 s / 10 s
- `ON_CHANGE`
- 对三类数据分别测：累积计数器、瞬时水位、离散状态

最后算出**保真度**：观测到的峰值 / 真实峰值。这张表就是「为什么分钟级轮询看不见 PFC 风暴」的量化版。

对应课程：[11 gNMI + OpenConfig 遥测](../../site/docs/04-telemetry/11-gnmi.md)。

## 2. 传输层说明（重要）

真 gNMI 是 `gRPC + Protobuf`。本 lab 用 **HTTP + SSE** 实现，原因只有一个：
零第三方依赖，`go run .` 就能跑。所以：

| 对齐了 | 没对齐 |
|---|---|
| 路径语法（`[name=*]`、`...` 递归通配、前缀匹配） | 传输层（HTTP/1.1 + SSE ≠ gRPC/HTTP2） |
| 订阅模式语义（`SAMPLE` / `ON_CHANGE`） | 编码（JSON ≠ Protobuf） |
| 计数器累积语义、时间戳 | `Capabilities` / `Set` 的完整字段 |
| 三类数据的采集策略差异 | 断线重订阅、TLS、认证 |

**真 gnmic 连不上它。** 想接真客户端，两条路：

1. 拿一台真交换机（或 SONiC 容器）按课程里的 `gnmic subscribe` 命令跑
2. 用 `openconfig/gnmic` 的 mock target 模式，把本 lab 的路径树搬过去

学到的东西可以直接迁移：路径怎么设计、哪类数据用哪种模式、间隔开到多少会丢信息。

## 3. 跑起来

需要 Go 1.22 以上，无第三方依赖。

```bash
cd labs/06-gnmi-mock
make run      # 跑保真度对照（默认行为，秒级完成）
make serve    # 只起 mock 设备，监听 :9339
make paths    # 列出模拟设备支持的全部叶子路径
make test     # 11 个单测
```

用 curl 观察（另开一个终端）。**注意 `-g`**：路径里的 `[name=...]` 会被 curl 当成 URL glob，
不加 `-g` 请求会直接卡住，什么都不返回。

```bash
curl -s  'http://localhost:9339/capabilities'
curl -sg 'http://localhost:9339/get?path=/interfaces/interface[name=Ethernet0]/state/oper-status'
curl -sgN 'http://localhost:9339/subscribe?path=/interfaces/.../state/counters/out-pfc-pause-frames&mode=sample&interval=1s&duration=20s'
```

不加 `-serve` 也能单跑一次：

```bash
go run . -get '/interfaces/.../state/counters/out-octets'
go run . -subscribe '/qos/.../state/watermark-bytes' -mode sample -interval 500ms
```

## 4. 指标树

```
/interfaces/interface[name=Ethernet0|Ethernet4]/
├── state/oper-status                                    state
└── state/counters/
    ├── out-octets                                       counter
    ├── out-discards                                     counter
    ├── out-pfc-pause-frames                             counter
    └── ecn-marked-pkts                                  counter

/qos/interfaces/interface[id=Ethernet0]/
└── output/queues/queue[name=3]/
    └── state/
        ├── watermark-bytes                              gauge
        └── transmit-pkts                                counter
```

队列 `3` 是承载 RoCE 无损流量的优先级，对应课程里「按优先级的 pause 计数」。

被模拟的信号（每 20 s 一个周期）：

| 事件 | 形态 | 真实峰值 |
|---|---|---|
| PFC 风暴 | 10 s 开始，持续 1.5 s，400 帧/s | 400 帧/s |
| 队列水位 | 同窗口冲高，末段贴 200 KB 平台 | 200 KB |
| 接口抖动 | 每 7 s 一次，每次 300 ms DOWN | 20 s 内 3 次 |

## 5. 三张保真度表

### ① 累积计数器：`out-pfc-pause-frames`

```
模式            间隔    样本数   观测峰值帧/s   保真度    数据量
SAMPLE        100ms     201        400        100%   28.9 KB
SAMPLE           1s      21        400        100%    3.0 KB
SAMPLE           5s       5        120         30%    734 B
SAMPLE          10s       3         60         15%    439 B
ON_CHANGE     100ms      16        400        100%    2.3 KB
ON_CHANGE        1s       3        200         50%    441 B
ON_CHANGE        5s       2         40         10%    292 B
ON_CHANGE       10s       2         30          8%    292 B
```

三件事：

1. **1 s 采样刚好能分辨 1.5 s 的事件**。事件持续时长是分辨率的下限，不是可选项。
2. **10 s 采样把 400 帧/s 摊成 60 帧/s**。计数器没丢数据，丢的是时间分辨率 —— 一减一除，尖峰变平均。
3. **`ON_CHANGE` 用在计数器上是错的**。值不变就不推，平稳期一条样本都没有。1 s 检查间隔下，风暴期只推 1～2 次，速率算出来只有一半。

### ② 瞬时水位：`watermark-bytes`

```
模式            间隔    样本数   观测峰值KB   保真度    数据量
SAMPLE        100ms     201        200      100%   32.3 KB
SAMPLE           1s      21        172       86%    3.4 KB
SAMPLE           5s       5         32       16%    820 B
SAMPLE          10s       3         32       16%    491 B
```

水位是 gauge，单点就能判读，所以问题不是「算速率」而是「撞不撞得上」。
5 s 采样完全错过了 200 KB 的尖峰，看到的永远是 32 KB 稳态 —— **面板全绿，队列已经贴到 XOFF 两次了。**

这条最符合课程里那句话：*轮询看到的只是平均值永远正常*。

### ③ 离散状态：`oper-status`

```
模式            间隔    样本数   DOWN次数   召回率    数据量
SAMPLE        100ms     201        3      100%   25.4 KB
SAMPLE           1s      21        3      100%    2.7 KB
SAMPLE           5s       5        1       33%    647 B
ON_CHANGE     100ms       7        3      100%    910 B
ON_CHANGE        1s       7        3      100%    910 B
```

状态量是 `ON_CHANGE` 的主场：**召回率 100%，数据量省 28 倍**（910 B vs 25.4 KB）。
300 ms 的抖动，5 s 采样只能碰到 1 次，剩下两次连痕迹都没留。

反过来说，用 `SAMPLE` 采状态量是纯粹的浪费：99% 的样本写着「还是 UP」。

## 6. 分级订阅策略

课程里那张表的机器版，把上面的数字带进去就能算账：

| 数据类别 | 模式 | 间隔 | 依据 |
|---|---|---|---|
| PFC pause、ECN 标记、队列水位 | SAMPLE | 1 s | 保真度 100%，1.5 s 事件刚好分辨 |
| 端口 octets / discards / CRC | SAMPLE | 10–30 s | 趋势分析够用，数据量大 |
| 端口 up/down、邻居、告警状态 | **ON_CHANGE** | — | 100% 召回，省 28 倍带宽 |
| 排障临时深挖 | POLL | 按需 | 不占常驻带宽 |

算一笔数据量：一万台设备 × 每台 500 条路径 × 1 s = 500 万点/秒。
按本次实测的单点约 144 字节算，就是 **720 MB/s**。分级订阅不是优化，是可行性前提。

## 7. 参数

| 参数 | 默认 | 说明 |
|---|---|---|
| `-serve` | false | 只起 mock 设备 |
| `-addr` | `:9339` | `-serve` 的监听地址 |
| `-speed` | 60 | 虚拟时间加速倍率 |
| `-duration` | 20s | 订阅的虚拟时长，默认一个完整风暴周期 |
| `-paths` | false | 列出全部叶子路径 |
| `-get <path>` | 空 | 单次 Get |
| `-subscribe <path>` | 空 | 单次订阅，配合 `-mode` / `-interval` |
| `-bench` | true | 跑保真度对照（默认行为） |

## 8. 思考题

1. 把 `-duration` 改成 5 s 再跑，所有保真度都归零。原因是窗口里根本没有风暴事件
   （风暴从第 10 s 开始）。这说明一件事：**调采样策略之前，先确认观测窗口覆盖了事件周期**，
   否则你归因给「采样丢信息」的，其实是「窗口没事件」。
2. 1 s `SAMPLE` 采计数器保真度 100%。如果风暴从 1.5 s 缩到 300 ms 呢？
   改 `signal.go` 里的 `StormLen` 再跑一遍，1 s 还能抓住吗？
3. `ON_CHANGE` 采计数器时，平稳期一条样本都没有。这会让下游的什么操作变难？
   （提示：Grafana 上那条线会怎么画）
4. 三张表里 `ON_CHANGE` 的数据量都比 `SAMPLE` 小。为什么课程里还说
   「队列水线用 SAMPLE 而不是 ON_CHANGE」？

## 9. 模型边界

- **传输层不是 gRPC**：见第 2 节，真客户端连不上
- **信号是解析式**：没有真实的队列随机性、没有多流叠加、没有时钟抖动
- **单设备单队列**：不做跨设备关联，不做时钟对齐演示（那是 10-rca 的一半）
- **只有两个接口、一个队列**：路径树规模远小于真机
- **没有背压与重连**：订阅端读得慢不会影响服务端，断线不自动重订阅
- **`fast` 模式不睡**：对照实验靠它把 20 s 压缩成毫秒，代价是它不再是真流式

## 10. 两条坑

**curl 的 URL globbing。** 路径里的 `[name=Ethernet0]` 会被 curl 当成范围展开，请求发不出去，
终端里一片空白，看上去像服务端没响应。`-g` 关掉 globbing 就好。

这条坑值得记住，因为 gNMI 路径天生带方括号，几乎所有 curl 示例都要加 `-g`。


**`ON_CHANGE` 的「间隔」不是推送间隔，是检查间隔。**很多人以为它意味着「变化立刻推」，
于是把间隔设成 5 s，结果变成了「每 5 秒看一眼，变了才推」。

这个差别在状态量上无所谓（状态变得慢），在计数器上直接要命：1 s 检查间隔下，
计数的变化被攒成一批一批，算出来的速率只有真实值的一半。

真机上 `ON_CHANGE` 是「值一变就推」，用在累积计数器上等于每条报文都推一次，会直接把
采集链路打爆。所以真机根本不该这么配 —— 这是配置错误，不是性能取舍。

本 lab 用「周期检查 + 去重」模拟它，于是看到的是「检查间隔决定保真度」。那 50% 是模拟
方式的副作用，不是 gNMI 的语义。**模型边界要写清楚，不然就成了假故障。**
