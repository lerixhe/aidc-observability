#!/usr/bin/env python3
"""标题制式体检：核对每课 frontmatter 的 title 是否守住「第NN讲 · 名词条目」。

用法：
    python3 scripts/check-titles.py            # 全站体检（CI 跑这个）
    python3 scripts/check-titles.py --list     # 只把当前标题按文件顺序列一遍

退出码：0 全部合规；1 有不合规的标题。

规则依据 docs/DECISIONS.md D17，写作细则见 corpus/style/COURSE-STYLE.md §2.3：

1. 整数课号即讲次：`01-lab-setup.md` → `第01讲 · 实验环境搭建`
2. 插在中间的独立课挂「附」：`04b-adaptive-routing.md` → `第04讲附 · ...`
3. 00 篇不编号，只叫「开篇词」
4. 附录用「附录 A · 术语表」
5. 条目是名词短语：不带问号、不带破折号、不写「五分钟」这类卖点修饰
6. 整条 ≤22 字（实测中位 16、最长 21）

为什么讲次要从文件名反推：04b / 10b / 20b 是后加的，讲次一旦和文件名漂开，
读者照着「第五讲」找不到 `/01-paradigm/04b-adaptive-routing`，锚点与外链全错位。
"""

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
DOCS = ROOT / "site" / "docs"

MAX_LEN = 22
# 课号：两位数字，可选 b 后缀（04b、10b、20b 这类后插课）
LESSON = re.compile(r"^(\d{2})(b?)-")
TITLE_LINE = re.compile(r"^title:\s*(.+)$", re.M)
# 卖点式修饰：时长、数量、比喻开场的常见标记
SELL = ("五分钟", "一秒", "十条", "三招", "一句话")


def expected_prefix(rel: str) -> tuple[str, str]:
    """从相对路径反推期望的标题前缀，返回（前缀, 说明）。"""
    parts = Path(rel).parts
    name = parts[-1]

    if parts[0] == "appendix":
        m = re.match(r"^(\d{2})-", name)
        letter = chr(ord("A") + int(m.group(1)) - 1) if m else "?"
        return f"附录 {letter} · ", f"附录第 {m.group(1) if m else '?'} 个"

    m = LESSON.match(name)
    if not m:
        return "", "文件名没有课号前缀"
    num, b = m.groups()
    if num == "00":
        return "", "00 篇不编号"
    return f"第{num}讲{'附' if b else ''} · ", f"课号 {num}{b}"


def main() -> int:
    argv = sys.argv[1:]
    only_list = "--list" in argv

    files = sorted(p for p in DOCS.rglob("*.md") if p.name != "index.md")
    if not files:
        print(f"check-titles.py：{DOCS} 下没有 md", file=sys.stderr)
        return 1

    bad = []
    for path in files:
        rel = path.relative_to(DOCS).as_posix()
        text = path.read_text(encoding="utf-8")
        m = TITLE_LINE.search(text[:2000])
        title = m.group(1).strip().strip("\"'") if m else ""

        if only_list:
            print(f"{rel:<45}{title or '（缺 title）'}")
            continue

        if not title:
            bad.append((rel, "缺 frontmatter title"))
            continue

        prefix, why = expected_prefix(rel)
        if prefix and not title.startswith(prefix):
            bad.append((rel, f"应以「{prefix}」开头（{why}），实为「{title}」"))
            continue
        if not prefix and rel.startswith("00-intro/00-") and title != "开篇词":
            bad.append((rel, f"00 篇应只叫「开篇词」，实为「{title}」"))
            continue

        item = title[len(prefix):] if prefix else title
        if not item:
            bad.append((rel, "标题条目为空"))
            continue
        if "？" in title or "?" in title:
            bad.append((rel, f"条目不写问句：「{title}」"))
            continue
        if "——" in title:
            bad.append((rel, f"条目不写两段式破折号：「{title}」"))
            continue
        if any(w in item for w in SELL):
            bad.append((rel, f"条目删卖点修饰（{'/'.join(SELL)}）：「{title}」"))
            continue
        if len(title) > MAX_LEN:
            bad.append((rel, f"{len(title)} 字超上限 {MAX_LEN}：「{title}」"))
            continue

    if only_list:
        return 0

    if bad:
        print("❌ 标题制式不合规：")
        for rel, msg in bad:
            print(f"  {rel:<45}{msg}")
        print(f"\n共 {len(bad)} 处。规则见 COURSE-STYLE §2.3 与 DECISIONS D17。")
        return 1

    titles = []
    for path in files:
        m = TITLE_LINE.search(path.read_text(encoding="utf-8")[:2000])
        if m:
            titles.append(len(m.group(1).strip().strip("\"'")))
    mid = sorted(titles)[len(titles) // 2]
    print(f"✅ 标题制式合规：{len(files)} 个文件 · 中位 {mid} 字 · 最长 {max(titles)} 字")
    return 0


if __name__ == "__main__":
    sys.exit(main())
