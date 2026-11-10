#!/usr/bin/env python3
"""列出超长句与含分号的行，便于逐课精修。

用法：
    python3 scripts/long-sents.py site/src/content/docs/01-paradigm/02-why-different.mdx
    python3 scripts/long-sents.py --semi-only <file>   # 只列分号行
"""

import re
import sys

TRANS = ("也就是", "相当于", "等价于", "类比", "本质上")


def clean(text: str) -> str:
    text = re.sub(r"^---.*?\n---\n", "", text, flags=re.S)
    text = re.sub(r"```.*?```", "", text, flags=re.S)
    text = re.sub(r"^\s*\|.*$", "", text, flags=re.M)
    text = re.sub(r"^\s*import .*$", "", text, flags=re.M)
    return text


def strip_md(s: str) -> str:
    return re.sub(r"[*#>\-\s`]", "", s)


def main() -> int:
    args = [a for a in sys.argv[1:] if not a.startswith("-")]
    semi_only = "--semi-only" in sys.argv
    for path in args:
        raw = open(path, encoding="utf8").read()
        body = clean(raw)
        print(f"\n### {path.split('/')[-1]}")
        if semi_only:
            for i, line in enumerate(raw.split("\n"), 1):
                if "；" in line:
                    print(f"{i:>4}| {line}")
            continue
        hits = []
        for line in body.split("\n"):
            if not line.strip():
                continue
            for s in re.split(r"(?<=[。！？])", line):
                n = len(strip_md(s))
                if n > 60:
                    hits.append((n, s.strip()))
        for n, s in sorted(hits, reverse=True):
            print(f"{n:>4}字| {s}")
        tr = sum(body.count(w) for w in TRANS)
        print(f"--- 翻译词 {tr} 处；分号正文 {body.count('；')} 处")
    return 0


if __name__ == "__main__":
    sys.exit(main())
