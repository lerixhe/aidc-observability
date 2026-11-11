#!/usr/bin/env python3
"""列出超长句与含分号的行，便于逐课精修。

用法：
    python3 scripts/long-sents.py                                  # 全站
    python3 scripts/long-sents.py site/docs/01-paradigm/02-why-different.md
    python3 scripts/long-sents.py site/docs/02-lossless             # 整个目录递归
    python3 scripts/long-sents.py --semi-only site/docs/01-paradigm # 只列分号行

入参规则与 style-check.py 完全一致，共用 scripts/mdpaths.py：
单文件、目录、通配符都能传，不存在的路径直接报错退出（码 2）。
"""

import re
import sys

import mdpaths

TRANS = ("也就是", "相当于", "等价于", "类比", "本质上")


def clean(text: str) -> str:
    text = re.sub(r"^---.*?\n---\n", "", text, flags=re.S)
    text = re.sub(r"```.*?```", "", text, flags=re.S)
    text = re.sub(r"^\s*\|.*$", "", text, flags=re.M)
    text = re.sub(r"^\s*import .*$", "", text, flags=re.M)
    text = re.sub(r"\n## 参考资料.*?(?=\n## |\Z)", "\n", text, flags=re.S)
    return text


def strip_md(s: str) -> str:
    return re.sub(r"[*#>\-\s`]", "", s)


def main() -> int:
    files = mdpaths.collect_or_exit(sys.argv[1:], mdpaths.DEFAULT_DOCS, prog="long-sents.py")
    semi_only = "--semi-only" in sys.argv

    total_hits = 0
    for path in files:
        try:
            raw = open(path, encoding="utf8").read()
        except OSError as e:
            print(f"long-sents.py：读不了 {path}：{e}", file=sys.stderr)
            return 2
        body = clean(raw)

        if semi_only:
            lines = [(i, line) for i, line in enumerate(raw.split("\n"), 1) if "；" in line]
            if not lines:
                continue
            print(f"\n### {path}")
            for i, line in lines:
                print(f"{i:>4}| {line}")
                total_hits += 1
            continue

        hits = []
        for line in body.split("\n"):
            if not line.strip():
                continue
            for s in re.split(r"(?<=[。！？])", line):
                n = len(strip_md(s))
                if n > 60:
                    hits.append((n, s.strip()))
        tr = sum(body.count(w) for w in TRANS)
        semi = body.count("；")
        if not hits and not tr and not semi:
            continue

        print(f"\n### {path}")
        for n, s in sorted(hits, reverse=True):
            print(f"{n:>4}字| {s}")
        print(f"--- 翻译词 {tr} 处；分号正文 {semi} 处")
        total_hits += len(hits) + tr + semi

    print(f"\n共 {len(files)} 个文件，{total_hits} 处待看。")
    return 0


if __name__ == "__main__":
    sys.exit(main())
