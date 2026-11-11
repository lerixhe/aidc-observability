#!/usr/bin/env python3
"""Mermaid 语法体检：扫 Markdown 里的 ```mermaid 块，静态校验图定义。

用法：
    python3 scripts/check-mermaid.py                    # 扫全仓（课程、lab README、.mmd）
    python3 scripts/check-mermaid.py labs/README.md     # 单文件
    python3 scripts/check-mermaid.py diagram.mmd        # 纯 .mmd 文件
    python3 scripts/check-mermaid.py --render           # 另用 mmdc 真渲染（需 npm i -g @mermaid-js/mermaid-cli）
    python3 scripts/check-mermaid.py --list             # 只列块，不校验

退出码：0 通过；1 有错；2 入参或环境有问题。

为什么不全靠渲染器：mmdc 要拖一个 Chromium，CI 里装一次几十秒到几分钟。
静态校验能拦住绝大多数真错 —— 类型关键字写错、括号不配对、引号没闭合、
linkStyle 指向不存在的边号。真渲染留成 --render，本地偶尔跑一次。

校验项：
    1. 首行是已知图类型（flowchart / graph / sequenceDiagram / classDiagram /
       stateDiagram-v2 / erDiagram / gantt / pie / journey / mindmap / timeline …）
    2. 圆括号、方括号、花括号成对，双引号成对
    3. flowchart 里的 linkStyle 序号必须小于边数（这条最容易在生成代码时踩）
    4. 节点标签里出现未转义的 <br> 之外尖括号时给出提示
"""

from __future__ import annotations

import argparse
import os
import re
import shutil
import subprocess
import sys
import tempfile

import mdpaths

# 已知的 Mermaid 图类型关键字（v10/v11 文档里的顶层声明）
DIAGRAM_TYPES = (
    "flowchart", "graph", "sequenceDiagram", "classDiagram", "classDiagram-v2",
    "stateDiagram", "stateDiagram-v2", "erDiagram", "journey", "gantt", "pie",
    "quadrantChart", "requirementDiagram", "gitGraph", "C4Context", "mindmap",
    "timeline", "zenuml", "sankey-beta", "xychart-beta", "block-beta",
    "packet-beta", "architecture-beta", "kanban",
)

FENCE = re.compile(r"```mermaid[ \t]*\n(.*?)```", re.S)
EDGE_RE = re.compile(r"(-->|---|-\.->|==>|--[xo]|-->|\-\-\|)")

# 默认扫这几处：课程正文、lab 说明（GitHub 渲染 mermaid）、规划文档、单独的 .mmd
DEFAULT_ROOTS = ("site/docs", "labs", "docs", ".")


def blocks_of(path: str):
    """产出 (起始行号, 内容)。行号用于报错定位。

    `.mmd` 文件整份当作一个块；Markdown 按 ```mermaid 围栏提取。
    """
    text = open(path, encoding="utf-8").read()
    if path.endswith(".mmd"):
        yield 1, text
        return
    for m in FENCE.finditer(text):
        line = text[: m.start()].count("\n") + 2  # 块内第一行的真实行号
        yield line, m.group(1)


def first_keyword(body: str) -> str:
    for line in body.splitlines():
        s = line.strip()
        if not s or s.startswith("%%"):
            continue
        return s.split()[0].rstrip(";")
    return ""


def balanced(body: str, open_ch: str, close_ch: str) -> bool:
    depth = 0
    in_quote = False
    for ch in body:
        if ch == '"':
            in_quote = not in_quote
            continue
        if in_quote:
            continue
        if ch == open_ch:
            depth += 1
        elif ch == close_ch:
            depth -= 1
            if depth < 0:
                return False
    return depth == 0 and not in_quote


def count_edges(body: str) -> int:
    n = 0
    for line in body.splitlines():
        s = line.strip()
        if not s or s.startswith(("%%", "subgraph", "end", "classDef", "class ", "style ", "linkStyle")):
            continue
        if EDGE_RE.search(s):
            n += len(EDGE_RE.findall(s))
    return n


def check_block(body: str) -> list[str]:
    errs = []
    kw = first_keyword(body)
    if not kw:
        return ["块是空的，只有 ```mermaid 与 ```"]

    if kw not in DIAGRAM_TYPES:
        errs.append(f"首行关键字 {kw!r} 不是已知图类型，可用：flowchart / sequenceDiagram / …")

    if kw in ("flowchart", "graph"):
        if not re.match(r"^(flowchart|graph)\s+(TD|TB|BT|LR|RL)\b", body.strip().split("\n")[0].strip()):
            head = body.strip().split("\n")[0].strip()
            errs.append(f"方向声明不规范：{head!r}，应为 '{kw} TD' 这类")

    for o, c in (("(", ")"), ("[", "]"), ("{", "}")):
        if not balanced(body, o, c):
            errs.append(f"括号 {o}{c} 不配对或双引号未闭合")

    if body.count('"') % 2:
        errs.append("双引号个数为奇数，有一段标签没有闭合")

    # linkStyle 越界：生成图时最常犯的错，序号从 0 开始，等于边数即越界
    edges = count_edges(body)
    for m in re.finditer(r"^\s*linkStyle\s+([\d,\s]+)", body, re.M):
        for part in m.group(1).split(","):
            part = part.strip()
            if not part:
                continue
            if not part.isdigit():
                continue
            if int(part) >= edges:
                errs.append(f"linkStyle {part} 越界：本图只有 {edges} 条边（合法序号 0–{max(edges - 1, 0)}）")

    # 未转义尖括号（<br/> 之外基本都会吃掉 HTML 渲染）
    for m in re.finditer(r"<(?!/?br\s*/?>)(/?)([a-z][\w-]*)", body):
        errs.append(f"标签里出现未预期的 HTML 标签 <{m.group(1)}{m.group(2)}>，Mermaid 会当 HTML 解析")
        break

    return errs


def render_body(body: str, mmdc: str, tmpdir: str) -> str:
    src = os.path.join(tmpdir, "in.mmd")
    out = os.path.join(tmpdir, "out.svg")
    with open(src, "w", encoding="utf-8") as f:
        f.write(body)
    try:
        r = subprocess.run(
            [mmdc, "-i", src, "-o", out, "--quiet"],
            capture_output=True, text=True, timeout=120,
        )
    except subprocess.TimeoutExpired:
        return "mmdc 渲染超时（120s）"
    if r.returncode != 0:
        msg = (r.stderr or r.stdout or "").strip().splitlines()
        return "mmdc 渲染失败：" + (msg[0] if msg else f"退出码 {r.returncode}")
    return ""


def main() -> int:
    ap = argparse.ArgumentParser(add_help=True)
    ap.add_argument("paths", nargs="*", help="文件或目录，默认 site/docs")
    ap.add_argument("--render", action="store_true", help="用 mmdc 真渲染每个块（慢）")
    ap.add_argument("--list", action="store_true", help="只列出块，不校验")
    args = ap.parse_args()

    if args.paths:
        files = mdpaths.collect_or_exit(args.paths, None, prog="check-mermaid.py", exts=(".md", ".mmd"))
    else:
        files = mdpaths.collect_many(list(DEFAULT_ROOTS), exts=(".md", ".mmd"))
        if not files:
            print("check-mermaid.py：默认目录里没找到可扫的文件", file=sys.stderr)
            return 2

    mmdc = ""
    if args.render:
        mmdc = shutil.which("mmdc") or ""
        if not mmdc:
            print("check-mermaid.py：加了 --render 却找不到 mmdc。"
                  "安装：npm i -g @mermaid-js/mermaid-cli", file=sys.stderr)
            return 2

    total = bad_blocks = 0
    bad_files = 0
    tmpdir = tempfile.mkdtemp(prefix="mermaid-")
    try:
        for path in files:
            file_errs = []
            for line, body in blocks_of(path):
                total += 1
                if args.list:
                    print(f"{path}:{line}  {first_keyword(body) or '(空)'}")
                    continue
                errs = check_block(body)
                if args.render and not errs:
                    msg = render_body(body, mmdc, tmpdir)
                    if msg:
                        errs.append(msg)
                for e in errs:
                    file_errs.append(f"{path}:{line}  {e}")
            if file_errs:
                bad_files += 1
                bad_blocks += len({x.split()[0] for x in file_errs})
                print(f"\n❌ {path}")
                for e in file_errs:
                    print(f"   {e}")
    finally:
        shutil.rmtree(tmpdir, ignore_errors=True)

    if args.list:
        print(f"\n共 {total} 个 mermaid 块。")
        return 0

    if total == 0:
        print("没有找到 mermaid 块。")
        return 0

    if bad_files:
        print(f"\n❌ {bad_files} 个文件里的 mermaid 块有问题（共 {total} 块）")
        return 1

    mode = "静态 + mmdc 渲染" if args.render else "静态"
    print(f"✅ Mermaid 语法通过（{mode}）：{len(files)} 个文件 · {total} 个块")
    return 0


if __name__ == "__main__":
    sys.exit(main())
