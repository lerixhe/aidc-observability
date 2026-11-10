# scripts/

| 脚本 | 状态 | 用途 |
|---|---|---|
| `check-labs.sh` | 可用（v0.2） | labs 统一检查：每个 lab 过 `go vet` + `go test`，再核对 `expected/` 与 `samples/` |
| `check-links.py` | 可用（v1.0） | 链接体检：站内路由、相对路径、锚点；`--external` 加查外链 |
| `style-check.py` | 可用（v0.2，v1.0 分档） | 文风体检：COURSE-STYLE 的硬项阻塞，观察项只打印 |
| `check-sensitive.sh` | 可用 | 脱敏扫描：检查 `site/` 下是否混入敏感词 |
| `html2mdx.py` | 可用（v0.1） | 解析原型 HTML，产出 MDX 骨架（只做机械转换，语义改写人工完成） |
| `normalize-quotes.py` | 可用 | 引号统一为 `""`（COURSE-STYLE §10-D） |
| `long-sents.py` | 可用 | 列出超长句，配合 §10-B 改造 |
| `sensitive-words.txt` | 本地，不入库 | 敏感词表，从 `.example` 复制后按需扩充 |

## 用法

```bash
# 首次：准备词表
cp scripts/sensitive-words.txt.example scripts/sensitive-words.txt

# 迁移后扫描
./scripts/check-sensitive.sh

# labs 检查（CI 也跑这个）
./scripts/check-labs.sh

# 链接体检（CI 只跑站内；外链手工跑，结果里的「拒绝访问」是噪声不是断链）
python3 scripts/check-links.py
python3 scripts/check-links.py --external --verbose

# 文风体检（硬项不达标退出码 1）
python3 scripts/style-check.py
python3 scripts/style-check.py --files

# 单课转换（原型 → MDX 骨架，输出到 /tmp 校对后再手工重写）
python3 scripts/html2mdx.py reference/archive-aidc/lesson-01.html -o /tmp/l1.mdx

# 覆盖标题（用 docs/OUTLINE.md 里的新课标题）
python3 scripts/html2mdx.py reference/archive-aidc/lesson-03.html \
  --title "RoCEv2：把 RDMA 搬到以太网" -o /tmp/l3.mdx
```

`html2mdx.py` 的产出**不能直接入库**：

- 只做结构映射（详情见 [../docs/MIGRATION.md](../docs/MIGRATION.md)「结构映射表」）
- 原型的 `div.compare` 双栏会转成表格，行语义需人工核对（产出里带 `{/* MIGRATION */}` 标记）
- 面试语境、雇主名、个人履历原样保留 → 必须人工改写
- 每课固定的 7 段结构（本节目标 / 机制拆解 / …）缺哪段，脚本会在「待人工处理」里列出来

`check-labs.sh` 会重新生成 `expected/` 再 `git diff`。改了模型就有 diff，
确认是有意改动后 `git add labs/<lab>/expected` 一起提交。
它由 `.github/workflows/labs.yml` 在 `labs/**` 变更时触发。

`check-links.py` 与 `style-check.py` 由 `.github/workflows/docs.yml` 在内容变更时触发。
链接体检只查站内：外链会因对方反爬（ACM、DOI 常见 403）与网络抖动误报，
所以判定分两级 —— 404/410/域名解析失败算断链，403/429/拒连只记警告。

`check-sensitive.sh` 在 CI 中可选启用（词表不进库，故 CI 用内置的最小词表）。
