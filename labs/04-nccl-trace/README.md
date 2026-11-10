# 04-nccl-trace · 从日志还原流量矩阵

> **教学级模型，非硬件行为等价。** 日志是生成的，行格式是本 lab 精简过的。
> 解析思路是真的，字段要按你自己 NCCL 版本的输出调整。

## 1. 这个 lab 做什么

拿一份 NCCL 通信日志，做三件事：

1. **还原流量矩阵**：谁发给谁、多少字节、每步多久
2. **核算带宽**：algbw → busbw，和端口线速对账
3. **定位 straggler**：哪一步慢、慢多少、归因到哪个 rank

对应课程：[08 NCCL 与集合通信](../../site/docs/03-traffic-topo/08-nccl.md)。

这门课的核心观念之一是「流量矩阵已知」。这个 lab 就是把它变成可算的东西：
**从日志反推出本该已知的账本，然后拿实际值去对账。**

## 2. 跑起来

需要 Go 1.22 以上，无第三方依赖。

```bash
cd labs/04-nccl-trace
make run     # 解析 samples/nccl-ring8.log 并打印核算结论
make gen     # 重新生成样本日志
make test    # 10 个单测，覆盖解析、反推环、straggler 归因
```

## 3. 日志格式

样本 `samples/nccl-ring8.log` 长这样：

```text
[0] NCCL INFO nranks 8 rank 0 comm 0x7f0000000000
[0] NCCL INFO Ring 00 : 0 -> 2 -> 4 -> 6 -> 1 -> 3 -> 5 -> 7 -> 0
[0] NCCL INFO opCount 1 allReduce bytes 16777216
[0] NCCL TRACE step 1 opCount 1 rank 0 peer 2 bytes 29360128 dur_us 1266
```

四类行，各管一件事：

| 行 | 提供什么 | 缺失时的降级 |
|---|---|---|
| `nranks N rank R` | rank 总数 | 从 rank / peer 编号反推最大值 +1 |
| `Ring NN : a -> b -> ...` | 环顺序 | **从发送关系自动反推**（`RebuildRing`） |
| `opCount K allReduce bytes B` | 每步消息大小 | 从流量反推：总发送量 ÷ 2(N-1) |
| `TRACE step S rank R peer P bytes B dur_us D` | 流量矩阵与时序 | 没有它就没得算 |

真实环境采集：

```bash
NCCL_DEBUG=INFO NCCL_DEBUG_SUBSYS=INIT,COLL,TRACE \
  NCCL_DEBUG_FILE=/tmp/nccl.%h.%p.log ./train.sh
```

不同 NCCL 版本的 `TRACE` 行字段会变，改 `parse.go` 里那四条正则就行。
解析器已经能处理 ANSI 转义码和无法识别的噪声行。

## 4. 看什么

```text
逐步核算（同步通信，步耗时 = 最慢 rank）
step         耗时ms    algbw    busbw   最慢rank  判定
1            1.27     13.3     23.2        0  正常
7            2.74      6.1     10.7        3  慢 2.2×
8            2.79      6.0     10.5        3  慢 2.2×

基线 busbw 23.1 GB/s（端口 25 GB/s，效率 92%）
异常步 7,8,9 · 最慢 rank 集中在 3（3 步里 3 步）
多花 37 卡·毫秒，全部是等待时间
```

四个要点：

1. **algbw 与 busbw 差 1.75 倍**（N=8）：`busbw = algbw × 2(N-1)/N`。
   报带宽不写清是哪个口径，等于没说。NCCL 自己打的是 busbw。
2. **步耗时取最慢 rank**：同步通信里，等最慢的等于全员一起等。
   单看某个 rank 的曲线，你会以为它有 1.2 ms 的空闲，其实它在等别人。
3. **基线取中位数**：10 步里有 3 步故障，用平均值会被拉低，故障就不显眼了。
4. **卡·毫秒**：`(实际耗时 - 基线耗时) × rank 数`。一次 3 步的 straggler 在万卡集群上
   就是几十卡·秒的纯等待，这才是「网络故障代价」的计价单位。

流量矩阵长这样：

```text
src\dst       0       1       2       3       4       5       6       7
     0       ·       ·   280.0       ·       ·       ·       ·       ·
     1       ·       ·       ·   280.0       ·       ·       ·       ·
     2       ·       ·       ·       ·   280.0       ·       ·       ·
```

一个 rank 只有一个下游，是环的定义。8 个 rank 每步各发 280 MB
（= 2 × 7/8 × 16 MB × 10 步），矩阵稀疏且规律。

## 5. 三个实验

### 实验一：把 Ring 行删掉

```bash
grep -v "Ring 00" samples/nccl-ring8.log > /tmp/noring.log
go run . -log /tmp/noring.log
```

输出里那句会变成「日志无 Ring 行，已从发送关系反推」，结论一字不差。

**这是本 lab 最值得记住的一步**：环顺序本来就是流量矩阵的属性。
只要拿到「谁发给谁」，就拿到了编排。这也是「流量矩阵已知 → 可以做预期 vs 实际比对」
能落地的前提。

### 实验二：换一个环顺序

```bash
go run . -gen /tmp/spine.log -ring 0,4,1,5,2,6,3,7
go run . -log /tmp/spine.log
```

环顺序变了，流量矩阵跟着变，busbw 基本不变（同样 8 个 rank、同样数据量）。
**Ring 只决定哪些链路被打满，不改变算法带宽。**
这句话在拓扑选型时的分量，跑一遍就有感觉。

### 实验三：造一个更长的 straggler

```bash
go run . -gen /tmp/long.log -steps 40 -fault 10-19,5,3.0
go run . -log /tmp/long.log
```

10 步、3 倍慢、40 步的日志。看「异常步」和「卡·毫秒」两个数字怎么随故障时长增长。

### 实验四：让故障占到多数，看基线怎么瞎

```bash
go run . -gen /tmp/dark.log -steps 40 -fault 10-30,5,3.0
go run . -log /tmp/dark.log
```

31 步里有 21 步是慢的。结果会很反直觉：

```text
基线 busbw 9.4 GB/s（端口 25 GB/s，效率 37%）
没有检测到异常步
⚠ 步耗时呈二峰分布（P25 8.2 / P75 23.5 GB/s）：慢步可能占了多数，
  中位数基线已被污染。此时应改用 P75 或历史基线做参照。
```

**拖久了，慢就变成了新的正常。** 中位数基线被多数派污染，异常检测直接失效。
这就是为什么真实的 SLO 要用历史基线或固定目标，不能就地取中位数。

这行警告是刻意加的：工具必须能说出「我不知道」。
如果它一声不响地报「没有异常」，你反而更该怀疑它。

## 6. 参数

| 参数 | 默认 | 说明 |
|---|---|---|
| `-log` | samples/nccl-ring8.log | 要解析的日志 |
| `-port` | 25 | 单端口线速 GB/s，算效率用 |
| `-gen <path>` | 空 | 生成日志到指定路径后退出 |
| `-ranks` | 8 | 生成时的 rank 数 |
| `-steps` | 10 | 生成时的步数 |
| `-msg-mb` | 16 | 每步 allReduce 消息大小 |
| `-ring` | 拓扑感知顺序 | 自定义环顺序 |
| `-fault` | 7-9,3,2.5 | 注入 straggler：步区间,rank,倍数 |
| `-no-fault` | false | 生成干净日志 |

## 7. 思考题

1. 日志里 `peer` 只有一个值，是因为 Ring 算法里每个 rank 只跟一个下游通信。
   换成 Tree 算法，这个矩阵会长什么样？稀疏度怎么变？
2. 基线取中位数。如果故障持续了 8 步里的 7 步，中位数会被污染成什么？
   这时该换用什么基线？
3. 「多花 37 卡·毫秒」按 8 rank 算。真实万卡集群上同样的故障比例，
   这个数字是多少？这个数字该记在网络的账上还是训练的账上？
4. 如果日志里某一步缺了 2 个 rank 的 TRACE 行，你的核算会偏多少？
   解析器现在的行为是什么？（提示：`Analyze` 里按步聚合，没做完整性校验）

## 8. 模型边界

- **日志是生成器造的**：环、耗时、抖动都由 `gen.go` 决定，真实 NCCL 的输出字段更多更杂
- **只有 AllReduce + Ring**：没有 Tree / CollNet / NVLS，也没有 ReduceScatter 与 AllGather 拆分
- **没有并发流**：真实一个 rank 上同时在跑多个 channel，本 lab 只算聚合值
- **没有网卡与链路层面的信息**：看不到 PFC、ECN、重传，那些在 02-pfc-sim 里
- **不做完整性校验**：缺 rank、缺步只会在结果里体现成偏差，不会报错

## 9. 一条坑

`dur_us` 的对象搞错会很自然。日志里每个 rank 都有一条记录，但**步耗时不是它们的和，
也不是平均，是最大值**。同步通信里谁都不能先走一步。

早期版本按平均值算，busbw 漂亮地停在 23 GB/s，straggler 完全看不出来。
换成 max 之后，一次 2.5 倍的慢立刻在时间线上跳出来。判读口径错了，指标再精确也没用。
