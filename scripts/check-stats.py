#!/usr/bin/env python3
"""规模数字体检：核对文档里对外宣称的「几课 / 几个实验 / 几条」与文件系统实数是否一致。

用法：
    python3 scripts/check-stats.py            # 只查门面文件里的规模声明（CI 跑这个）
    python3 scripts/check-stats.py --all      # 打印全部实数，人工对照

退出码：0 全部对得上；1 有声明与实数不符。

为什么只查少数几处：正文里的「3 课」「07 课」「前 20 课」指的是模块课时或课号，
不是规模总数，全文正则误报率高于收益。这里只锚定四个门面文件里的状态行，
和附录里两条能被机器数出来的清单长度。数字真变了，改这几处就够。
"""

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
DOCS = ROOT / "site" / "docs"
LABS = ROOT / "labs"
APPENDIX = DOCS / "appendix"

# 模块目录：两位数字前缀，appendix / public 不算
MODULE_DIR = re.compile(r"^\d{2}-")

# 需要 Docker + containerlab 的实验才带这文件，用它区分 L1 与 L2
L2_MARKER = "topo.clab.yml"


def module_dirs():
    if not DOCS.is_dir():
        return []
    return [p for p in sorted(DOCS.iterdir()) if p.is_dir() and MODULE_DIR.match(p.name)]


def count_lessons(module: Path) -> int:
    return sum(1 for f in module.glob("*.md") if f.name != "index.md")


# 实验目录：两位数字前缀，如 00-setup
LAB_DIR = re.compile(r"^\d{2}-")

def lab_dirs():
    if not LABS.is_dir():
        return []
    return [p for p in sorted(LABS.iterdir()) if p.is_dir() and LAB_DIR.match(p.name)]


def count_glossary() -> int:
    """术语表：按主题分成若干张表，只数数据行，跳过表头行。"""
    path = APPENDIX / "01-glossary.md"
    if not path.is_file():
        return 0
    n = 0
    for line in path.read_text(encoding="utf-8").splitlines():
        if not line.startswith("| ") or line.startswith("| 术语"):
            continue
        if set(line) <= set("|-: "):  # 分隔行
            continue
        n += 1
    return n


def count_references() -> int:
    """参考资料：每条都以 Markdown 链接开头，数这个最稳。"""
    path = APPENDIX / "02-references.md"
    if not path.is_file():
        return 0
    return sum(1 for line in path.read_text(encoding="utf-8").splitlines() if line.startswith("| ["))


def actual_stats():
    modules = module_dirs()
    labs = lab_dirs()
    l2 = [p for p in labs if (p / L2_MARKER).exists()]
    return {
        "lessons": sum(count_lessons(m) for m in modules),
        "modules": len(modules),
        "labs": len(labs),
        "l1": len(labs) - len(l2),
        "l2": len(l2),
        "glossary": count_glossary(),
        "references": count_references(),
    }


# (文件, 锚点正则, 实数键, 人话说明)
# 正则只放一个捕获组，锚点取得足够长，避免命中正文里的课号与课时数
CHECKS = [
    ("README.md", r"(\d+) 课 \+ 附录", "lessons", "课程数"),
    ("README.md", r"(\d+) 个实验全部落地", "labs", "实验数"),
    ("README.md", r"(\d+) 个 L1 \+ ", "l1", "L1 实验数"),
    ("README.md", r"\+ (\d+) 个 L2", "l2", "L2 实验数"),
    ("README.en.md", r"(\d+) lessons \+", "lessons", "课程数"),
    ("README.en.md", r"(\d+) labs shipped", "labs", "实验数"),
    ("README.en.md", r"\((\d+) L1 \+ ", "l1", "L1 实验数"),
    ("README.en.md", r"L1 \+ (\d+) L2", "l2", "L2 实验数"),
    ("site/README.md", r"(\d+) 课 \+ 附录", "lessons", "课程数"),
    ("site/README.md", r"(\d+) 个实验落地", "labs", "实验数"),
    ("labs/README.md", r"(\d+) 个 lab 全部落地", "labs", "实验数"),
    ("labs/README.md", r"L1 层 (\d+) 个", "l1", "L1 实验数"),
    ("docs/PROJECT.md", r"(\d+) 个 L1 lab", "l1", "L1 实验数"),
    ("docs/PROJECT.md", r"(\d+) 个 L2 lab", "l2", "L2 实验数"),
    ("docs/OUTLINE.md", r"(\d+) 条中英对照", "glossary", "术语条数"),
    ("docs/OUTLINE.md", r"(\d+) 条，按可信度分层", "references", "参考条数"),
    ("docs/ANNOUNCE.md", r"(\d+) 条术语表", "glossary", "术语条数"),
    ("docs/ANNOUNCE.md", r"(\d+) 条参考资料", "references", "参考条数"),
    ("site/docs/appendix/02-references.md", r"本清单涉及的 (\d+) 条外部链接", "references", "参考条数"),
]


def main() -> int:
    stats = actual_stats()

    if "--all" in sys.argv:
        label = {
            "lessons": "课程",
            "modules": "模块",
            "labs": "实验",
            "l1": "L1 实验",
            "l2": "L2 实验",
            "glossary": "术语",
            "references": "参考资料",
        }
        print("实数（从文件系统数出来的）：")
        for k, v in stats.items():
            print(f"  {label[k]:<8}{v}")
        print()

    bad = []
    for rel, pattern, key, what in CHECKS:
        path = ROOT / rel
        if not path.is_file():
            bad.append((rel, what, "文件不存在", stats[key]))
            continue
        text = path.read_text(encoding="utf-8")
        m = re.search(pattern, text)
        if not m:
            # 锚点没命中：多半是文案被改写，提示一下但不当作错误
            print(f"跳过  {rel}  {what}：锚点未命中（文案可能改过，请更新本脚本）")
            continue
        claimed = int(m.group(1))
        if claimed != stats[key]:
            line = text[: m.start()].count("\n") + 1
            bad.append((rel, what, f"{rel}:{line} 写 {claimed}", stats[key]))

    if bad:
        print("❌ 规模数字与文件系统不符：")
        for rel, what, got, want in bad:
            print(f"  {what:<10}{got}  →  实为 {want}")
        print("\n改文案，或确认真数变了再改脚本里的锚点。")
        return 1

    print(f"✅ 规模数字一致：{stats['lessons']} 课 / {stats['labs']} 实验"
          f"（{stats['l1']} L1 + {stats['l2']} L2）/ {stats['glossary']} 条术语"
          f" / {stats['references']} 条参考")
    return 0


if __name__ == "__main__":
    sys.exit(main())
