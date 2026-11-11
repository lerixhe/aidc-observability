# 智算网络可观测指南 · AIDC Observability

> 从有损 IDC 到无损智算：机制、遥测与可运行的实验室。

**状态：v1.0** —— 23 课 + 附录 A/B，12 个实验全部落地（10 个 L1 + 2 个 L2 真拓扑）。
站点：<https://lerixhe.github.io/aidc-observability/> · [English README](README.en.md)

> 可读（v0.1）、可跑（v0.2）、可讲全（v0.3）、可进阶（v0.4）、可交付（v1.0）。
> v1.0 起课程结构冻结，只接受纠错类 PR（见 [DECISIONS.md](docs/DECISIONS.md) D9）。
> L2 拓扑实验走双模式：默认零依赖出剧本与模型结论，`make deploy` 才需要 containerlab。

---

## 这门课解决什么问题

AI 训练网络（智算中心、AIDC、AI Fabric，也有人叫高性能网络）把网络可观测的基本假设全部翻转了：

| | 传统 IDC | 智算中心 |
|---|---|---|
| 流量模型 | 南北向小流、随机到达 | 东西向大象流、AllReduce 周期性同步突发 |
| 丢包语义 | 背景噪声，TCP 静默重传 | 事故本身，恢复代价极大 |
| 故障代价 | 单机 / 单批请求 | 小时 × 千卡，回退半小时 |
| 遥测粒度 | 分钟级 SNMP 轮询 | 秒级 / 亚秒级流式订阅 |

把 IDC 的监控面板原样搬过去必然失效。本课程逐层拆解这套新规则：从 RDMA / RoCEv2
的因果链，到 PFC / ECN 的机制与失败模式，再到 gNMI / INT 遥测采集、指标体系与 RCA 自动化。

## 差异化

讲 RoCE / PFC 机制的资料很多，但**讲「可观测」这个横切视角**、并且
**能在容器里复现 PFC 风暴与 incast 的动手实验室**，几乎没有。

课件讲机制，`labs/` 给能跑起来复现的代码 —— 这是本项目想补的缺口。

## 目录结构

```
.
├── docs/                  规划与决策文档（先读 PROJECT.md）
│   ├── PROJECT.md         项目上下文：定位、读者、技术栈、里程碑、风险
│   ├── OUTLINE.md         23 课大纲 + 与课程原型的映射关系
│   ├── MIGRATION.md       原型迁移与去个人化/脱敏规则
│   └── DECISIONS.md       关键决策记录（ADR）
├── site/                  VitePress 文档站（v0.1 已上线，见 site/README.md）
├── labs/                  Go / 容器化实验代码（12 个 lab：10 L1 + 2 L2）
├── reference/             课程原型参考（本地用，不入库，见下）
└── scripts/               迁移脚本、链接体检、文风体检、脱敏扫描
```

## 路线图

| 版本 | 内容 | 状态 |
|---|---|---|
| v0.0 | 立项：规划文档、目录骨架、原型归档 | 已完成 |
| v0.1 | 内容上线：16 课迁移 Markdown + 站点 + GitHub Pages | 已完成 |
| v0.2 | 可运行：首批 6 个 L1 lab（无特权依赖）+ devcontainer + labs CI | 已完成 |
| v0.3 | 扩课：新增 8 篇（导读、环境、超节点、INT、管道、指标体系、MTTR、端到端案例）+ 3 个 L1 lab（07/08/09）+ 附录 A 术语表与 B 参考清单 | 已完成 |
| v0.4 | 进阶 lab：containerlab 真实拓扑（05-clos-topo）+ 故障注入（03-fault-inject）+ 端到端剧本（11-case-study） | 已完成 |
| v1.0 | 打磨：链接体检脚本 + Docs CI、双语 README、宣传稿、结构冻结 | **当前** |

详见 [docs/PROJECT.md](docs/PROJECT.md)。

## 关于 reference/

`reference/archive-aidc/` 是一套早期的私有课程原型（17 课自包含 HTML），
本项目的结构与教学法由它演化而来。

该目录**含个人化叙述与前雇主相关信息，已在 `.gitignore` 中排除，不纳入版本库**，
仅作本地迁移底本使用。公开内容一律按 [docs/MIGRATION.md](docs/MIGRATION.md)
脱敏重写后才进入 `site/`。

## 许可

- 文字与课程内容：[CC BY-SA 4.0](LICENSE)
- `labs/` 与 `scripts/` 代码：[MIT](LICENSE-CODE)

## 贡献

项目已到 v1.0，课程结构冻结。欢迎纠错类 Issue，请标注具体课程与来源依据：

- 机制类断言请附可验证来源（RFC / IEEE 标准 / 论文 / 厂商公开文档）
- 脱敏红线见 [docs/MIGRATION.md](docs/MIGRATION.md) 第三节，涉及内部信息的改动不会被合并
- 本地开发：`cd site && pnpm install && pnpm dev`
