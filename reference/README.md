# reference/ — 课程原型参考

## archive-aidc/（不入库）

一套早期的私有课程原型：17 课自包含 HTML（`index.html` + `lesson-01..17.html`），
外加原始共享样式 `assets/course.css`。

**本项目的教学结构、模块划分与「要点卡 / 折叠答案 / 双栏对比」等设计由它演化而来。**

- 用途：迁移底本（源材料），不对外发布
- 已加入 `.gitignore`，不纳入版本库
- 原因：含个人化叙述与前雇主相关信息，见 [../docs/MIGRATION.md](../docs/MIGRATION.md)

## 原型结构速览（供迁移参考）

```
archive-aidc/
├── index.html          总览：能力快照 / 课程目标 / 学习约定 / 课程表
├── lesson-01..17.html  每课：hero / card 小节 / compare 双栏 / link-exp 经验映射
│                       / quiz-item + details 折叠答案 / keycard 要点卡 / nav 前后课
└── assets/course.css   共享样式（另有 sync-style.py 做内联，未随附）
```

每课固定内容单元：

| 单元 | 原型实现 | 迁移目标 |
|---|---|---|
| 课头 | `header.hero` + `.lesson-no` | frontmatter `title` / `description` |
| 正文 | `div.card` + `h2` 分级小节 | `##` / `###` |
| 对比 | `div.compare`（old/new 双栏） | Markdown 表格 |
| 经验映射 | `div.link-exp` | `::: info 经验映射` |
| 警示 | `div.warn` | `::: warning` |
| 练习 | 2 必答 + 2 自测，`<details>` 折叠答案 | `::: details` 折叠，改「参考答案 + 评分维度」 |
| 要点卡 | `div.keycard`（深色块） | `::: tip 本课要点` |
| 导航 | `div.nav` 上下课 | 删除，用站点内置分页 |

原型课目一览（→ 新编号见 [../docs/OUTLINE.md](../docs/OUTLINE.md)）：

| 原型 | 标题 |
|---|---|
| L1 | 四差异：流量模型 / 丢包语义 / 故障代价 / 遥测粒度 |
| L2 | RDMA：AI 训练为什么绕过内核 |
| L3 | RoCEv2：把 RDMA 搬到以太网上 |
| L4 | PFC：无损网络怎么做到不丢包 |
| L5 | ECN + DCQCN：光不丢包还不够 |
| L6 | 无损网络四大典型故障 |
| L7 | 模块一自测（原「毕业拷问」） |
| L8 | NCCL 与集合通信 |
| L9 | 万卡集群拓扑 |
| L10 | gNMI 与 OpenConfig 遥测 |
| L11 | 光模块：AIDC 故障第一来源 |
| L12 | IB vs RoCE 路线之争 |
| L13 | AIOps：告警收敛、RCA 与图计算 |
| L14 | LLM 运维 Agent 落地方式 |
| L15 | 某内部数仓 vs ClickHouse |
| L16 | 自测 + 能力迁移对照表（原「毕业拷问第二轮」） |
| L17 | 课程收尾 |

> 若本目录为空：原型尚未从原项目复制进来，或已被清理。请从原项目
> `find_job/course/archive-aidc/` 复制后使用。
