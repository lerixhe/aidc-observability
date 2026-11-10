#!/usr/bin/env python3
"""引号口径统一：把「」换成中文弯引号 “”，与 COURSE-STYLE §10-D 对齐。

只动代码块外的正文，代码块内保持原样（那里可能是命令输出或注释）。
用法：
    python3 scripts/normalize-quotes.py --dry-run    # 只报数
    python3 scripts/normalize-quotes.py             # 就地替换
"""

import glob
import sys

DOCS = "site/src/content/docs"


def convert(path: str) -> tuple[str, int]:
    lines = open(path, encoding="utf8").read().split("\n")
    in_fence = False
    hits = 0
    for i, line in enumerate(lines):
        if line.lstrip().startswith("```"):
            in_fence = not in_fence
            continue
        if in_fence or "「" not in line and "」" not in line:
            continue
        hits += line.count("「") + line.count("」")
        lines[i] = line.replace("「", "“").replace("」", "”")
    return "\n".join(lines), hits


def main() -> int:
    dry = "--dry-run" in sys.argv
    total = 0
    for f in sorted(glob.glob(f"{DOCS}/**/*.mdx", recursive=True)):
        text, hits = convert(f)
        if not hits:
            continue
        total += hits
        print(f"{f.split('/')[-1]:<24} {hits}")
        if not dry:
            open(f, "w", encoding="utf8").write(text)
    print(f"\n合计 {total} 处{'（dry-run，未写入）' if dry else '，已替换'}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
