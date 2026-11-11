# 语料库（corpus）

> 收集作者本人的**口癖、语言习惯、文字风格**，产出一份可操作的文风档案，
> 供后续课程材料、公众号与技术分享统一语体使用。
>
> 与 `reference/` 的区别：`reference/archive-aidc/` 是**内容底本**（讲什么），
> 这里是**语体底本**（怎么讲）。

---

## 目录结构

```
corpus/
├── README.md                      本文件：用途、流程、维护规则
├── sources/                       原始语料存档（原文保真，逐篇一份，不入库）
│   └── 2026-02-27-api-to-ui.md    《从API到UI：AI助力后端工程师全栈交付》
└── style/                         提炼产物
    ├── COURSE-STYLE.md           ★ 课程文风规则（依据 v2 特征卡，十条硬指标 + 禁用清单 + 检查脚本）
    ├── NARRATIVE-DIGEST-tcp-book.md  样本二：《深入理解 TCP 协议》前 3 篇的叙事手法拆解（管结构，不管句法）
    ├── STYLE.md                  v0.1 单篇样本稿（已被 v2 修正，仅作历史记录）
    └── LEXICON.md                口癖与用语词表（含课程禁用标记 ⛔）
```

配套工具：`scripts/style-check.py`（量化报表，超标退出码 1）。

## 依据分层

| 层 | 文件 | 地位 |
|---|---|---|
| 上游：写作特征卡 | `/Users/Lerix/Projects/corpus/writing-profile.md`（v2，203 篇 / 118,099 汉字） | 唯一数据依据 |
| 本库产出 | `style/COURSE-STYLE.md` | **写课程的唯一标准** |
| 辅证 | `style/LEXICON.md` | 查词，按 ⛔ 过滤 |
| 叙事参考 | `style/NARRATIVE-DIGEST-tcp-book.md` | 结构层手法（开头结尾同一句 / 先跑命令再讲理 / 跳过要说清楚），与 COURSE-STYLE 冲突时以后者为准 |
| 历史 | `style/STYLE.md` | 已被 v2 推翻，归档用 |

## 怎么用

| 场景 | 动作 |
|---|---|
| 写新课程 / 新文章 | 读 `style/COURSE-STYLE.md`，成稿后跑 `python3 scripts/style-check.py` |
| 改写他人初稿为本人口吻 | 打开 `style/LEXICON.md`，逐条替换起句词、转折词、结论词（⛔ 条目除外） |
| 让 AI 代笔 | 投喂 `COURSE-STYLE.md` + `LEXICON.md`，并声明"不要写 §9 黑名单句式和 AI 味胶水词" |
| 课程稿移植 | 文风按 `COURSE-STYLE.md`，脱敏按 `docs/MIGRATION.md`，脱敏优先 |

## 新增语料的流程

1. 原文存入 `sources/`，命名 `YYYY-MM-DD-<slug>.md`，**保留原样不编辑**（错字也是风格证据）。
2. 在 `style/LEXICON.md` 里给命中的词条累加出处（`S1`、`S2`…）。
3. 只有**跨 2 篇以上复现**的特征才升级进 `STYLE.md`；单篇孤例留在 LEXICON 的第 11 节待验证区。
4. 更新 `STYLE.md` 的版本号与「置信度」一节。

## 归档规则

- `sources/` 中的原文可能含内部平台名、厂商痕迹、内部链接（截图域名等），
  **已在 `.gitignore` 中排除**，仅本地保留。
- `style/` 下的提炼文件可入库，但引用例句时必须先脱敏（平台名泛化、链接删除、规模数字改量级）。

## 当前状态

| 项 | 值 |
|---|---|
| 版本 | v2.0（对齐 writing-profile v2） |
| 样本 | 上游 203 篇 / 118,099 汉字；本地存档 1 篇 |
| 本库结论 | 分句层已达标；整句 P90 69（目标 ≤ 40）、分号 8.4%（目标 ≤ 0.5%）、翻译词 0 处/课（目标 ≥ 3） |
| 下一步 | 按 `COURSE-STYLE.md` §10 改造 A→B→C 项，优先清分号与长句，补翻译式解释 |
