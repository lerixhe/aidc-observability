# 项目上下文（总档案）

> 本文件是项目的单一事实来源。清理上下文后、执行任何动作前，先读本文件，
> 再读 [OUTLINE.md](OUTLINE.md)（课程结构）与 [DECISIONS.md](DECISIONS.md)（已定决策）。
>
> 最后更新：v1.x（23 课 + 附录 A/B 冻结；站点底层由 Astro Starlight 切到 VitePress）。

---

## 一、项目定义

| 项 | 决策 |
|---|---|
| 仓库名 | `aidc-observability` |
| 中文名 | 智算网络可观测指南 |
| 副标题 | 从有损 IDC 到无损智算：机制、遥测与可运行的实验室 |
| 许可证 | 内容 CC BY-SA 4.0 / 代码 MIT（双许可） |
| 站点框架 | VitePress + GitHub Pages |
| 默认语言 | 中文（英文站推迟到 v2，README 双语：`README.md` / `README.en.md`） |
| 当前版本 | v2.0（26 课；10 个 L1 lab + 2 个 L2 lab；Docs CI 五项检查 + 产物锚点爬虫） |

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
- 「拷问课」改为「篇末自测卷」

---

## 二、技术栈

| 层 | 选型 | 理由 |
|---|---|---|
| 文档站 | VitePress 1.x（Vue 3） | 内置本地搜索、Markdown 单源、纯静态产出、PR diff 可读、CI 简单 |
| 部署 | GitHub Pages + `pnpm build` + `upload-pages-artifact` | 零成本、零服务器 |
| 部署分支 | `gh-pages`（Actions 推送） | 保持 `main` 干净 |
| 内容格式 | 纯 Markdown + 容器语法（`::: tip` / `::: details`） | 降低贡献门槛，不需要 MDX 编译链 |
| Lab 语言 | Go（主）+ Python（数据脚本）+ YAML（拓扑） | Go 与目标读者栈匹配，Python 便于数据处理 |
| Lab 分层 | L1 无门槛（`go run`）/ L2 需 Docker | 见第四节 |
| 开发环境 | `.devcontainer/` + GitHub Codespaces | 解决 macOS 跑容器网络的痛点 |
| 包管理 | pnpm（site）、go modules（labs） | — |

### VitePress 结构约定

- 内容根是 `site/docs/`（`srcDir`），配置与主题在 `site/.vitepress/`
- 目录带数字前缀（如 `01-paradigm/`），侧边栏在 `config.mts` 里按目录自动生成，新增课不改配置
- 原 `.card` → 普通小节；`hero` → frontmatter `title` + `description`
- `🔗 关联你的经验` → `::: info 经验映射`
- `.keycard` 要点卡 → `::: tip 本课要点`
- `.compare` 双栏对比 → Markdown 表格（移动端更友好）
- 折叠答案 → `::: details 参考答案与评分维度`（核心互动设计，**保留**）
- 手写前后课 nav → 删除，改用内置上一课/下一课分页
- 站内链接写 `.md` 相对路径，`cleanUrls: true` 产出无后缀 URL

---

## 三、课程内容

23 课 / 7 篇 + 2 附录，完整大纲与来源映射见 **[OUTLINE.md](OUTLINE.md)**。

- 开篇 · 认识这门课（2 课）
- 第一篇 · 两套世界观（3 课）
- 第二篇 · 不丢包的代价（3 课）
- 第三篇 · 流量与连线（3 课）
- 第四篇 · 把现场变成数据（4 课）
- 第五篇 · 从数据到结论（5 课）
- 第六篇 · 路线与收口（3 课）
- 附录 A 术语表 / B 参考资料

内容质量机制（强制）：

- 每课末尾必须有 `## 参考资料`，机制类断言需可验证来源（RFC 8926 RoCEv2、
  SIGCOMM'15 DCQCN、OpenConfig models、NCCL docs、厂商白皮书）
- 每课固定结构见 OUTLINE.md 的「每课统一结构」，课标题统一为「主标题 —— 副标题」两段式（D13）

---

## 四、Lab 分层（项目成败关键）

核心约束：PFC / ECN 是硬件行为，纯软件仿真无法做到行为等价。
若不分层，多数读者会在第一条 `docker` 命令处流失。

| 层 | 名称 | 技术要求 | 内容 |
|---|---|---|---|
| **L1 无门槛** | 数据面模拟 | 仅 Go / Python | `pfc-sim` 参数化模型生成 pause / ECN 时序；`gnmi-mock` 可订阅的 mock server；`pipeline` 本地落盘；`rca` 吃模拟告警跑图算法；`case-study` 端到端排障剧本与 MTTR 时间账 |
| **L2 进阶** | 真拓扑仿真 | Docker + Linux（推荐 Codespaces） | `clos-topo` containerlab 搭 2 spine / 4 leaf 量 ECMP 极化；`fault-inject` tc/netem 复现四类故障并给遥测指纹判读器 |

L2 的两个 lab 都是**双模式**（D8）：默认模式只生成剧本与模型结论，`go run` 即跑，
产出锁进 `expected/`；`make deploy` 才需要 containerlab。这样 CI 能校验判据，
读者也不会卡在第一条 `docker` 命令上。

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

| 版本 | 交付物 | 估时 |
|---|---|---|
| v0.0 立项 | 定名、双许可、目录骨架、规划文档、原型归档 | 0.5-1 天（已完成） |
| v0.1 上线 | 17 课迁移 Markdown + 去个人化 + 站点导航 + Pages 上线<br>（已完成：站点 ✅ / html2mdx.py ✅ / 16 课迁移 ✅ / Pages 上线 ✅） | 3-4 天（已完成） |
| v0.2 可跑 | 首批 L1 lab + `.devcontainer` + labs CI<br>（已完成：00-setup / 01-traffic-model / 02-pfc-sim / 04-nccl-trace / 06-gnmi-mock / 10-rca ✅ / devcontainer ✅ / labs CI ✅） | 4-6 天（已完成） |
| v0.3 扩课 | 新增 8 篇（00 导读、01 环境、10 超节点、12 INT、14 管道、15 指标体系、19 MTTR、21 端到端）+ 3 个 L1 lab<br>（已完成：✅ 8 篇 / ✅ 07-int-path / ✅ 08-pipeline / ✅ 09-slo / ✅ 站点收口） | 5-6 天（已完成） |
| v0.4 进阶 | containerlab 拓扑 + 故障注入 + 端到端剧本<br>（已完成：✅ 03-fault-inject / ✅ 05-clos-topo / ✅ 11-case-study / ✅ 双模式与 CI 校验 / ✅ 课程入口接线） | 5-7 天（已完成） |
| v1.0 打磨 | 全站链接体检脚本、双语 README、宣传稿<br>（已完成：✅ check-links.py / ✅ Docs CI / ✅ README.en.md / ✅ docs/ANNOUNCE.md / ✅ 文风口径分档 D10 / ✅ 结构冻结 D9） | 2-3 天（已完成） |
| v2.0 工具箱 | 3 篇前沿课（AR / Scale-Up NVLink / UEC）+ 遥测可视化栈 + 真实 0x8808 帧与 XDP 闸门 + Codespaces 预装 + CI 双保险<br>（已完成：✅ 10b / 04b / 20b / ✅ check-stats.py / ✅ /metrics + compose + 2 套看板 / ✅ Mermaid 与 DOT 导出 / ✅ containerlab 预装 / ✅ inject_pfc_pause.py + xdp_pfc.c / ✅ check-mermaid.py + check-anchors-html.py / ✅ mdpaths.py 统一入参） | 10 天（已完成） |

**发布策略：先发 v0.1 再补 lab。** 内容上线才有反馈；lab 是最重的部分，不要憋大招。
v1.0 冻结课程结构，只接受纠错类 PR；v2.0 依 D14 有限解冻，新增内容仍走 issue 讨论后开分支。

---

## 六、风险登记

| 风险 | 严重度 | 应对 |
|---|---|---|
| 内置本地搜索中文分词弱，站内搜索差 | 中 | VitePress local search（minisearch）分词偏字级；不满意再换 Algolia DocSearch（开源项目免费），待实测 |
| containerlab 在 macOS 需特权 / 虚拟机 | 高 | 主推 Codespaces + `.devcontainer`；macOS 文档指向 Lima / OrbStack；L2 的默认模式零依赖，缺 Docker 只降级不阻塞 |
| 软件仿真 PFC 的教学真实性受质疑 | 中 | 每个 lab 开头声明「非硬件等价」+ 给真机验证路径 |
| 机制细节准确性 | 高 | 每课 `参考资料`；PR 模板要求标注来源；设 `content-fix` issue 模板 |
| **脱敏不彻底 → 前雇主合规风险** | **高** | 迁移后跑「专有信息扫描」（公司名 / 内部系统名 / 真实数字 grep 白名单），见 MIGRATION.md |
| 23 课维护成本 | 中 | v1.0 冻结结构，新内容先 issue 讨论再开分支 |

---

## 七、待确认事项

- [x] 仓库名：`aidc-observability`（站点 `base` 已按此配置）
- [x] GitHub 账号：`lerixhe`（`gh` 已登录，repo 权限可用）；远端仓库尚未创建
- [x] L2 lab 是否接受「主推 GitHub Codespaces」：接受，但 L2 默认模式不依赖它（D8）
- [ ] v1.0 是否同步出英文站：否，英文站推迟到 v2，v1.0 只做双语 README
- [ ] 是否公开署名作者身份（影响脱敏尺度）
- [ ] P4 搜索方案最终态：v1.0 保留内置本地搜索，理由见下
  - 取舍：local search 零依赖、离线、随构建产出，换 DocSearch 要引入外部服务与密钥
  - 已知短板：中文按字切分，长词精确度一般，「无损网络」这类查询召回偏高
  - 触发换的条件：出现三次以上「搜不到某个具体名词」的读者反馈
  - 换的成本：加 Algolia 账号 + CI 索引步骤 + `.vitepress/config.mts` 装配，约半天
