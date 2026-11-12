#!/usr/bin/env python3
"""自查脚本：把临摹留下的原书作者自称（老钱）换成第一人称「笔者」。

背景：本课程的每一篇都对着原书的一段临帖写，落笔时难免把原作者的自称
一起带过来（「以老钱的经验」「老钱表示」「老钱不推荐」）。课程正文里出现
另一个人的名字，读者会以为这是转述而不是作者自己的判断，所以统一换成
第一人称。

只改正文，不碰代码：
  * fenced 代码块（``` / ~~~）整段跳过 —— 里面有 `laoqian` 这类示例数据
  * 行内代码（`...`）跳过 —— 指标名、命令、字段名都在里面
  * `reference/` 目录拒绝处理 —— 那是原书，一字不能动

用法：
  python3 scripts/replace-self-reference.py                 # 预览，不写盘
  python3 scripts/replace-self-reference.py --write         # 真改
  python3 scripts/replace-self-reference.py --check         # CI：有命中即失败
  python3 scripts/replace-self-reference.py site/docs/05-analysis
  python3 scripts/replace-self-reference.py --map 老钱式=笔者式
  python3 scripts/replace-self-reference.py --list-rules    # 看替换表

退出码：0 通过；1 有命中（dry-run 或 --check）；2 参数/环境错误。
"""

from __future__ import annotations

import argparse
import os
import re
import sys

# ---- 替换表：长词在前，避免「老钱式」被「老钱」先吃掉一半 ----
DEFAULT_RULES: list[tuple[str, str]] = [
    ("老钱的经验", "笔者的经验"),
    ("老钱式", "笔者式"),
    ("老钱", "笔者"),
]

# 这些地方一律不动
PROTECTED_DIRS = ("reference",)
CODE_FENCE = re.compile(r"^\s*(```|~~~)")
INLINE_CODE = re.compile(r"`[^`]*`")


def find_project_root(start: str) -> str:
    cur = os.path.abspath(start)
    while True:
        if os.path.isdir(os.path.join(cur, "site", "docs")):
            return cur
        parent = os.path.dirname(cur)
        if parent == cur:
            raise SystemExit("找不到项目根目录（应包含 site/docs）")
        cur = parent


def iter_markdown(targets: list[str]) -> list[str]:
    files: list[str] = []
    for target in targets:
        if os.path.isfile(target):
            if target.endswith(".md"):
                files.append(target)
            continue
        for root, dirs, names in os.walk(target):
            dirs[:] = [d for d in dirs if d not in {"node_modules", "dist", "dist-check", ".vitepress", "__pycache__", ".git"}]
            for name in sorted(names):
                if name.endswith(".md"):
                    files.append(os.path.join(root, name))
    return sorted(set(files))


def is_protected(path: str, root: str) -> bool:
    rel = os.path.relpath(os.path.abspath(path), root)
    return rel.split(os.sep)[0] in PROTECTED_DIRS


def mask_inline_code(line: str) -> tuple[str, list[str]]:
    """把行内代码换成占位符，替换完再还原。"""
    pockets: list[str] = []

    def swap(match: re.Match[str]) -> str:
        pockets.append(match.group(0))
        return f"\x00{len(pockets) - 1}\x00"

    return INLINE_CODE.sub(swap, line), pockets


def unmask(line: str, pockets: list[str]) -> str:
    def swap(match: re.Match[str]) -> str:
        return pockets[int(match.group(1))]

    return re.sub(r"\x00(\d+)\x00", swap, line)


def rewrite(text: str, rules: list[tuple[str, str]]) -> tuple[str, list[tuple[int, str, str, str]]]:
    """返回 (新文本, 命中列表)。命中项：(行号, 原文片段, 新片段, 整行上下文)。"""
    out_lines: list[str] = []
    hits: list[tuple[int, str, str, str]] = []
    in_fence = False
    fence_mark = ""

    for lineno, line in enumerate(text.splitlines(keepends=True), start=1):
        fence = CODE_FENCE.match(line)
        if fence:
            mark = fence.group(1)
            if not in_fence:
                in_fence, fence_mark = True, mark
            elif mark == fence_mark:
                in_fence = False
            out_lines.append(line)
            continue
        if in_fence:
            out_lines.append(line)
            continue

        masked, pockets = mask_inline_code(line)
        new_line = masked
        for old, new in rules:
            if old in new_line:
                hits.append((lineno, old, new, line.strip()))
                new_line = new_line.replace(old, new)
        out_lines.append(unmask(new_line, pockets))

    return "".join(out_lines), hits


def main(argv: list[str]) -> int:
    parser = argparse.ArgumentParser(
        description="把临摹留下的原书作者自称换成第一人称「笔者」",
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    parser.add_argument("paths", nargs="*", help="扫描路径，默认 site/docs")
    parser.add_argument("--write", action="store_true", help="写盘（默认只预览）")
    parser.add_argument("--check", action="store_true", help="CI 模式：有命中即退出 1")
    parser.add_argument("--map", action="append", default=[], metavar="旧=新", help="追加替换规则，可多次")
    parser.add_argument("--list-rules", action="store_true", help="打印替换表后退出")
    args = parser.parse_args(argv)

    rules = list(DEFAULT_RULES)
    for pair in args.map:
        if "=" not in pair:
            print(f"--map 需要 旧=新 的形式，收到：{pair}", file=sys.stderr)
            return 2
        old, new = pair.split("=", 1)
        if not old:
            print("--map 的旧词不能为空", file=sys.stderr)
            return 2
        rules.insert(0, (old, new))
    # 长词优先，避免短词先吃掉一部分
    rules.sort(key=lambda item: len(item[0]), reverse=True)

    if args.list_rules:
        for old, new in rules:
            print(f"{old} → {new}")
        return 0

    root = find_project_root(os.getcwd())
    targets = args.paths or [os.path.join(root, "site", "docs")]
    targets = [t if os.path.isabs(t) else os.path.join(os.getcwd(), t) for t in targets]

    for target in targets:
        if not os.path.exists(target):
            print(f"路径不存在：{target}", file=sys.stderr)
            return 2

    files = iter_markdown(targets)
    if not files:
        print("没有找到 Markdown 文件")
        return 0

    total_hits = 0
    changed_files = 0
    skipped: list[str] = []

    for path in files:
        if is_protected(path, root):
            skipped.append(os.path.relpath(path, root))
            continue
        with open(path, encoding="utf-8") as handle:
            text = handle.read()
        new_text, hits = rewrite(text, rules)
        if not hits:
            continue

        total_hits += len(hits)
        changed_files += 1
        rel = os.path.relpath(path, root)
        for lineno, old, new, context in hits:
            print(f"{rel}:{lineno}  {old} → {new}")
            print(f"    {context}")
        if args.write:
            with open(path, "w", encoding="utf-8") as handle:
                handle.write(new_text)

    print()
    if skipped:
        print(f"跳过受保护目录：{', '.join(skipped)}")
    if total_hits == 0:
        print("✅ 未发现需要替换的作者自称")
        return 0

    verb = "已改写" if args.write else "待改写"
    print(f"{verb} {total_hits} 处，涉及 {changed_files} 个文件")
    if not args.write:
        print("这是预览。确认无误后加 --write 落盘。")
        return 1 if args.check else 0
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
