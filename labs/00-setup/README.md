# 00-setup · 环境自检

> 这个 lab 不模拟任何网络行为，它只回答一个问题：**这台机器能跑哪些 lab。**

## 1. 干什么

一次跑完三件事：

1. **工具体检**：Go / Docker / containerlab / python3 是否可用、版本够不够
2. **lab 盘点**：`labs/` 下哪些已落地、各自需要什么、你这台机器能不能跑
3. **冒烟检查**（`-try`）：对每个可跑的 L1 lab 执行 `go vet` + `go test`

## 2. 跑起来

```bash
cd labs/00-setup
make run     # 体检报告
make try     # 体检 + 冒烟检查
make test    # 单测
```

不需要任何第三方依赖。

## 3. 输出长什么样

```text
工具               版本           状态     说明
go               1.26.3       ✅
Docker           —            ❌      未安装，或不在 PATH 里
containerlab     —            ❌      未安装，L2 lab 需要
python3          3.14.7       ✅

lab 盘点（4 个已落地 / 共登记 9 个）
  目录                 层    需要工具                 对应课      可跑     说明
  00-setup           L1   go                   01       ✅      环境自检：这台机器能跑哪些 lab
  01-traffic-model   L1   go                   02       ✅      IDC 小流 vs AllReduce 同步突发
  02-pfc-sim         L1   go                   05,06    ✅      PFC / ECN / DCQCN 参数化队列模型
  04-nccl-trace      L1   go                   08       ✅      从 NCCL 日志还原流量矩阵与 straggler
  03-fault-inject    L2   go+docker            07       —      规划中 · tc/netem 复现 HoL 与 incast
```

三个字段值得看一眼：

- **可跑**：`✅` 能跑，`❌` 缺工具，`—` 还没落地
- **层**：L1 只需要 Go；L2 需要 Docker，容器的东西都在 v0.4
- **缺什么**：`❌` 那行会写清缺哪个工具，不用自己猜

## 4. 依赖说明

| 工具 | 谁需要 | 没有会怎样 |
|---|---|---|
| **Go ≥ 1.22** | 所有 L1 lab | 跑不了，先装 <https://go.dev/dl/> |
| Docker | L2 lab（v0.4） | L1 全部照跑，不影响 |
| containerlab | L2 拓扑实验 | 同上 |
| python3 | `scripts/` 里的校验脚本 | 不影响 lab |

macOS 上没有 Docker 也好办：L1 的四个 lab 一个都不需要它。
真想跑 L2 拓扑，用 GitHub Codespaces 比在本机折腾虚拟机省事（见 `labs/README.md`）。

## 5. 参数

| 参数 | 说明 |
|---|---|
| `-try` | 额外跑冒烟检查（`go vet` + `go test`），有失败则退出码为 1 |
| `-root` | 指定仓库根目录，默认从当前目录向上找含 `labs/` 的位置 |

`-try` 在 CI 里当门禁用，参见 `.github/workflows/labs.yml`。

## 6. 加一个新 lab 时要改什么

`probe.go` 里有一张 `knownLabs` 登记表。新 lab 落地后补一行：

```go
{"07-int-path", "L1", []string{"go"}, "12", "INT 逐跳元数据的编码与解析", true},
```

漏了也不致命：自检会把它列进「未登记的目录」，提醒你补。

`Ready` 字段只是声明，实际以「目录里有没有 `go.mod`」为准。声明了却没代码，
仍然显示为未落地。

## 7. 思考题

1. 为什么这个自检要求「有 `go.mod` 才算落地」，而不是相信登记表里的 `Ready`？
   换成你维护 CI 的场景，哪种更省事、哪种更可靠？
2. L2 lab 需要 Docker，而 macOS 上 Docker 要跑虚拟机。这种情况下
   「本机可跑」的判断该不该把 CPU 架构也算进去？（Apple Silicon 跑 x86 镜像）
3. `-try` 只跑 `go vet` 和 `go test`，不跑 `make run`。省略试跑的风险是什么？

## 8. 边界

- 只探测「工具在不在、版本够不够」，不探测性能（CPU 主频、网卡、内存）
- 不检查 `site/` 那边的 Node / pnpm，那是文档站的事
- Docker daemon 的检查是靠 `docker info` 是否成功，不区分「没装」和「没启动」之外的细节
- 冒烟检查串行执行，labs 多了会很慢，没有并行

## 9. 一条坑

`docker --version` 成功**不代表能用**。常见情况是二进制装了、daemon 没起来
（macOS 上尤其常见，Docker Desktop / OrbStack 没启动）。

所以探测 Docker 时多跑了一次 `docker info`。只看 `--version` 的检查，
会在读者执行第一条 `docker run` 时才发现问题，那时候他已经以为是 lab 坏了。
