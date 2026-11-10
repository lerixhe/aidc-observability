# 项目上下文（总档案）

> 本文件是项目的单一事实来源。清理上下文后、执行任何动作前，先读本文件，
> 再读 [OUTLINE.md](OUTLINE.md)（课程结构）与 [DECISIONS.md](DECISIONS.md)（已定决策）。
>
> 最后更新：v0.1 内容迁移完成（16 课上线）。

---

## 一、项目定义

| 项 | 决策 |
|---|---|
| 仓库名 | `aidc-observability` |
| 中文名 | 智算网络可观测指南 |
| 副标题 | 从有损 IDC 到无损智算：机制、遥测与可运行的实验室 |
| 许可证 | 内容 CC BY-SA 4.0 / 代码 MIT（双许可） |
| 站点框架 | Astro Starlight + GitHub Pages |
| 默认语言 | 中文（英文站推迟到 v2，README 双语） |
| 当前版本 | v0.1（16 课已迁移上线；M0 导读、新增课与 lab 待后续里程碑） |

### 一句话定位

面向有传统 IDC 网络背景的工程师，讲清 AI 训练网络「看什么、为什么、怎么看」，
并给出可跑的采集、分析与故障复现 demo。

### 目标读者

1. 从 IDC / 云网络转智算网络的研发与 SRE
2. 智算网络监管控平台（可观测 / AIOps）的开发者
3. 想理解 RoCE 无损体系机制的后端与基础设施工程师

### 差异化（为什么值得开源）

- 讲 RoCE / PFC 机制的资料很多，**讲「可观测」这个横切视角的很少**
- **能在容器里复现 PFC 风暴 / incast / HoL 的动手实验室几乎没有**
- 多数中文资料停在概念层，缺「采集 → 指标 → 判读 → 定位」的完整链路

### 定位转换（关键）

课程原型是**求职面试突击**定位（含拷问课、面试语境、雇主名、薪资）。
开源版必须转为**工程学习路径**：读者不是「要过一面的你」，
而是「刚接手一个智算集群、需要知道该盯哪些指标的人」。

由此产生的硬性要求：

- 删除全部面试语境与「我了解某公司」类叙事
- 删除雇主专有名、内部系统名、真实架构与容量数字（见 MIGRATION.md）
- 练习题从「发给 AI 批改」改为「自答 + 对照评分维度自评」
- 「拷问课」改为「模块自测卷」

---

## 二、技术栈

| 层 | 选型 | 理由 |
|---|---|---|
| 文档站 | Astro 5 + Starlight | 内置 Pagefind 搜索、Markdown/MDX 单源、PR diff 可读、CI 简单 |
| 部署 | GitHub Pages + `withastro/action` | 零成本、零服务器 |
| 部署分支 | `gh-pages`（Actions 推送） | 保持 `main` 干净 |
| 内容格式 | Markdown + 少量 MDX（Aside / details） | 降低贡献门槛 |
| Lab 语言 | Go（主）+ Python（数据脚本）+ YAML（拓扑） | Go 与目标读者栈匹配，Python 便于数据处理 |
| Lab 分层 | L1 无门槛（`go run`）/ L2 需 Docker | 见第四节 |
| 开发环境 | `.devcontainer/` + GitHub Codespaces | 解决 macOS 跑容器网络的痛点 |
| 包管理 | pnpm（site）、go modules（labs） | — |

### Starlight 结构约定

- 目录带数字前缀（如 `01-paradigm/`），侧边栏用 `autogenerate`，新增课不改配置
- 原 `.card` → 普通小节；`hero` → frontmatter `title` + `description`
- `🔗 关联你的经验` → `<Aside type="note" title="经验映射">`
- `.keycard` 要点卡 → `<Aside type="tip" title="本课要点">`
- `.compare` 双栏对比 → Markdown 表格（移动端更友好）
- 折叠答案 → MDX 原生 `<details>`，**保留**（核心互动设计）
- 手写前后课 nav → 删除，改用 Starlight 内置 pagination

---

## 三、课程内容

24 课 / 6 模块 + 2 附录，完整大纲与来源映射见 **[OUTLINE.md](OUTLINE.md)**。

- M0 导读与实验室环境（2 课）
- M1 范式转换（3 课）
- M2 无损网络机制与失败模式（3 课）
- M3 流量与拓扑（3 课）
- M4 遥测数据面（4 课）
- M5 分析与自动化（5 课）
- M6 选型与收口（3 课）
- 附录 A 术语表 / B 参考资料

内容质量机制（强制）：

- 每课末尾必须有 `## 参考资料`，机制类断言需可验证来源（RFC 8926 RoCEv2、
  SIGCOMM'15 DCQCN、OpenConfig models、NCCL docs、厂商白皮书）
- 每课固定 6 段结构：**本节目标 → 机制拆解 → 经验映射 → 判读/实操 → 练习 → 要点卡**

---

## 四、Lab 分层（项目成败关键）

核心约束：PFC / ECN 是硬件行为，纯软件仿真无法做到行为等价。
若不分层，多数读者会在第一条 `docker` 命令处流失。

| 层 | 名称 | 技术要求 | 内容 |
|---|---|---|---|
| **L1 无门槛** | 数据面模拟 | 仅 Go / Python | `pfc-sim` 参数化模型生成 pause / ECN 时序；`gnmi-mock` 可订阅的 mock server；`pipeline` 本地落盘（DuckDB/SQLite）；`rca` 吃模拟告警跑图算法 |
| **L2 进阶** | 真拓扑仿真 | Docker + Linux（推荐 Codespaces） | `clos-topo` containerlab + FRR 搭 2 spine / 4 leaf；`fault-inject` tc/netem + 伪造 pause 帧复现 HoL 与 incast；`case-study` 完整排障剧本 |

每个 lab 统一结构：

```
labs/NN-<slug>/
├── README.md      # 目标 / 前置 / 命令 / 观察什么 / 思考题
├── Makefile       # make run / make clean
├── main.go 或 compose.yaml
└── expected/      # 预期输出样本，便于自校验
```

每个 L1 lab 开头必须声明：**「教学级模型，非硬件行为等价」**，并给出真机验证方法。

---

## 五、里程碑

| 阶段 | 交付物 | 估时 |
|---|---|---|
| M0 v0.0 立项 | 定名、双许可、目录骨架、规划文档、原型归档 | 0.5-1 天（已完成） |
| M1 v0.1 上线 | 17 课迁移 MDX + 去个人化 + Starlight 导航 + Pages 上线<br>（已完成：站点 ✅ / html2mdx.py ✅ / 16 课迁移 ✅ / Pages 上线 ✅） | 3-4 天（已完成） |
| M2 v0.2 可跑 | L1 层 4 个 lab + `.devcontainer` + labs CI<br>（已完成：00-setup / 01-traffic-model / 02-pfc-sim / 04-nccl-trace ✅ / devcontainer ✅ / labs CI ✅） | 4-6 天（已完成） |
| M3 v0.3 扩课 | 新增 7 课（导读、超节点、INT、遥测管道、指标体系、MTTR、端到端案例） | 5-6 天 |
| M4 v0.4 进阶 | containerlab 拓扑 + 故障注入 + 端到端剧本 | 5-7 天 |
| M5 v1.0 打磨 | 附录、术语表、链接检查、双语 README、宣传稿 | 2-3 天 |

**发布策略：先发 v0.1 再补 lab。** 内容上线才有反馈；lab 是最重的部分，不要憋大招。
v1.0 后冻结课程结构，只接受纠错类 PR。

---

## 六、风险登记

| 风险 | 严重度 | 应对 |
|---|---|---|
| Pagefind 中文分词弱，站内搜索差 | 中 | v0.1 先用 Pagefind；不满意则 v0.3 换 Algolia DocSearch（开源项目免费） |
| containerlab 在 macOS 需特权 / 虚拟机 | 高 | 主推 Codespaces + `.devcontainer`；macOS 文档指向 Lima / OrbStack |
| 软件仿真 PFC 的教学真实性受质疑 | 中 | 每个 lab 开头声明「非硬件等价」+ 给真机验证路径 |
| 机制细节准确性 | 高 | 每课 `参考资料`；PR 模板要求标注来源；设 `content-fix` issue 模板 |
| **脱敏不彻底 → 前雇主合规风险** | **高** | 迁移后跑「专有信息扫描」（公司名 / 内部系统名 / 真实数字 grep 白名单），见 MIGRATION.md |
| 24 课维护成本 | 中 | v1.0 冻结结构，新内容先 issue 讨论再开分支 |

---

## 七、待确认事项

- [x] 仓库名：`aidc-observability`（站点 `base` 已按此配置）
- [x] GitHub 账号：`lerixhe`（`gh` 已登录，repo 权限可用）；远端仓库尚未创建
- [ ] L2 lab 是否接受「主推 GitHub Codespaces」（引入免费额度依赖）
- [ ] v1.0 是否同步出英文站
- [ ] 是否公开署名作者身份（影响脱敏尺度）
