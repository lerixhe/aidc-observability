# scripts/

| 脚本 | 状态 | 用途 |
|---|---|---|
| `check-labs.sh` | 可用（v0.2） | labs 统一检查：每个 lab 过 `go vet` + `go test`，再核对 `expected/` 与 `samples/` |
| `check-links.py` | 可用（v1.0） | 链接体检：站内路由、相对路径、锚点（读源码）；`--external` 加查外链 |
| `check-anchors-html.py` | 可用（v2.0） | 产物爬虫：在 VitePress 构建出的 HTML 上验 `id` 与站内 `href` 是否真存在 |
| `check-mermaid.py` | 可用（v2.0） | Mermaid 语法体检：图类型、括号配对、`linkStyle` 越界；`--render` 走 mmdc 真渲染 |
| `check-stats.py` | 可用（v1.0） | 规模数字体检：门面文件里的几课 / 几个实验 / 几条，与文件系统实数比对 |
| `style-check.py` | 可用（v0.2，v1.0 分档，v2.0 统一入参） | 文风体检：COURSE-STYLE 的硬项阻塞，观察项只打印 |
| `check-titles.py` | 可用（v2.1） | 标题制式体检：从文件名反推讲次，校验前缀 / 问号 / 破折号 / 卖点修饰 / 长度 |
| `check-sensitive.sh` | 可用（v2.0 改 POSIX sh） | 脱敏扫描：检查 `site/` 下是否混入敏感词 |
| `mdpaths.py` | 可用（v2.0） | 共用入参模块：`style-check.py` 与 `long-sents.py` 的单文件 / 目录 / 通配符展开 |
| `html2mdx.py` | 可用（v0.1） | 解析原型 HTML，产出 Markdown 骨架（只做机械转换，语义改写人工完成） |
| `normalize-quotes.py` | 可用 | 引号统一为 `""`（COURSE-STYLE §10-D） |
| `long-sents.py` | 可用（v2.0 统一入参） | 列出超长句，配合 §10-B 改造 |
| `sensitive-words.txt` | 本地，不入库 | 敏感词表，从 `.example` 复制后按需扩充 |

## 共用入参规则

`style-check.py` 与 `long-sents.py` 的路径参数走同一份实现（`mdpaths.py`），行为一致：

```bash
python3 scripts/style-check.py                       # 默认 site/docs
python3 scripts/style-check.py site/docs/02-lossless # 目录递归取全部 .md
python3 scripts/style-check.py a.md b.md             # 多个文件
python3 scripts/style-check.py 'site/docs/**/*.md'   # 通配符
```

三条约定：

1. **不存在的路径直接报错，退出码 2。** 旧版会静默变成空列表，然后在汇总处抛
   `ZeroDivisionError` —— 报错位置离原因十米远。
2. **`index.md` 递归时跳过**（站点 Hero 版式，不属课程正文）；显式点名传进来时照常统计。
3. **重复参数自动去重**，`site/docs` 与 `site/docs/01-paradigm` 一起传不会算两遍。

## 用法

```bash
# 首次：准备词表
cp scripts/sensitive-words.txt.example scripts/sensitive-words.txt

# 迁移后扫描（POSIX sh，用 sh / dash 跑都成立）
./scripts/check-sensitive.sh
sh scripts/check-sensitive.sh site labs

# labs 检查（CI 也跑这个）
./scripts/check-labs.sh

# 链接体检（CI 只跑站内；外链手工跑，结果里的「拒绝访问」是噪声不是断链）
python3 scripts/check-links.py
python3 scripts/check-links.py --external --verbose

# 构建产物锚点爬虫（先构建）
cd site && pnpm build && cd ..
python3 scripts/check-anchors-html.py --verbose

# Mermaid 语法体检
python3 scripts/check-mermaid.py                    # 全仓 Markdown + .mmd
python3 scripts/check-mermaid.py --list             # 只列块与图类型
python3 scripts/check-mermaid.py --render           # 用 mmdc 真渲染（需 npm i -g @mermaid-js/mermaid-cli）
cd labs/05-clos-topo && go run . -exp 1 -format mermaid > /tmp/clos.md && cd ../..
python3 scripts/check-mermaid.py /tmp/clos.md       # 校验生成型图

# 文风体检（硬项不达标退出码 1）
python3 scripts/style-check.py
python3 scripts/style-check.py --files              # 每课一行，汇总红项仍生效

# 规模数字体检（CI 也跑这个；--all 先看一遍实数）
python3 scripts/check-stats.py
python3 scripts/check-stats.py --all

# 标题制式体检（COURSE-STYLE §2.3 / D17）
python3 scripts/check-titles.py                     # 全站，不合规退出码 1
python3 scripts/check-titles.py --list              # 按文件顺序把当前标题列一遍

# 单课转换（原型 → Markdown 骨架，输出到 /tmp 校对后再手工重写）
python3 scripts/html2mdx.py reference/archive-aidc/lesson-01.html -o /tmp/l1.md

# 覆盖标题（用 docs/OUTLINE.md 里的新课标题）
python3 scripts/html2mdx.py reference/archive-aidc/lesson-03.html \
  --title "第04讲 · RoCEv2 与 DCB" -o /tmp/l3.md
```

`html2mdx.py` 的产出**不能直接入库**：

- 只做结构映射（详情见 [../docs/MIGRATION.md](../docs/MIGRATION.md)「结构映射表」）
- 原型的 `div.compare` 双栏会转成表格，行语义需人工核对（产出里带 `<!-- MIGRATION -->` 标记）
- 面试语境、雇主名、个人履历原样保留 → 必须人工改写
- 每课固定的 7 段结构（本节目标 / 机制拆解 / …）缺哪段，脚本会在「待人工处理」里列出来

`check-labs.sh` 会重新生成 `expected/` 再 `git diff`。改了模型就有 diff，
确认是有意改动后 `git add labs/<lab>/expected` 一起提交。
它由 `.github/workflows/labs.yml` 在 `labs/**` 变更时触发。

`check-links.py`、`style-check.py`、`check-titles.py`、`check-mermaid.py`、`check-stats.py` 由
`.github/workflows/docs.yml` 在内容变更时触发。
链接体检只查站内：外链会因对方反爬（ACM、DOI 常见 403）与网络抖动误报，
所以判定分两级 —— 404/410/域名解析失败算断链，403/429/拒连只记警告。

`check-sensitive.sh` 在 CI 中可选启用（词表不进库，故 CI 用内置的最小词表）。

## 两套锚点检查为什么要并存

| | `check-links.py` | `check-anchors-html.py` |
|---|---|---|
| 输入 | Markdown 源码 | 构建后的 HTML |
| slug 来源 | 按 markdown-it-anchor 规则**近似还原** | 渲染器写好的 `id="..."` |
| 何时红 | PR 阶段，不用构建 | 构建之后 |
| 抓不到 | 插件生成的锚点、渲染期改写的 id | 源码里已经写错但被渲染修掉的链接 |

VitePress 构建自己只拦死路由，**死锚点它不管**。所以两边都要跑：
源码侧早发现，产物侧兜底。

## Mermaid 检查的边界

静态校验拦的是真错：图类型关键字拼错、括号或引号没闭合、`linkStyle` 序号越界、
标签里混进未转义的 HTML 标签。它拦不了「语法合法但图很难看」这类问题。

要真渲染就加 `--render`，它调 `mmdc`（`@mermaid-js/mermaid-cli`）。CI 里不跑：
装一次要拖 Chromium，几十秒到几分钟，收益不抵成本。本地偶尔跑一次足够。
