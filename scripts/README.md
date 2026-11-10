# scripts/

| 脚本 | 状态 | 用途 |
|---|---|---|
| `check-sensitive.sh` | 可用 | 脱敏扫描：检查 `site/` 下是否混入敏感词 |
| `html2mdx.py` | 可用（v0.1） | 解析原型 HTML，产出 MDX 骨架（只做机械转换，语义改写人工完成）；跑完会打印「待人工处理清单」 |
| `check-links.sh` | v1.0 实现 | 全站外链与站内锚点检查 |
| `sensitive-words.txt` | 本地，不入库 | 敏感词表，从 `.example` 复制后按需扩充 |

## 用法

```bash
# 首次：准备词表
cp scripts/sensitive-words.txt.example scripts/sensitive-words.txt

# 迁移后扫描
./scripts/check-sensitive.sh

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

`check-sensitive.sh` 在 CI 中可选启用（词表不进库，故 CI 用内置的最小词表）。
