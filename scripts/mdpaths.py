#!/usr/bin/env python3
"""Markdown 路径收集：check-mermaid.py 用的入参逻辑。

一套规则解决三件事，避免脚本各写一份、行为还悄悄不一致：

1. **单个文件 / 目录 / 混着传** 都支持。目录递归取 `*.md`。
2. **不存在的路径** 立刻报错退出（退出码 2），而不是静默变成空列表后
   在汇总处抛 ZeroDivisionError。旧版的行为就是后者。
3. **去重与稳定排序**：`site/docs` 和 `site/docs/01-paradigm` 一起传不会重复统计。

额外约定：
    `index.md` 是站点的 Hero 版式，不属于课程正文，递归时跳过；
    显式点名传进来的 `index.md` 仍然会统计 —— 用户要的就是它。
"""

from __future__ import annotations

import glob
import os
import sys

DEFAULT_DOCS = "site/docs"
SKIP_NAMES = {"index.md"}
# 递归时直接剪掉的目录：依赖、构建产物、缓存。
# 不用 glob 是因为它在 npm 式扁平的 node_modules 下会扫几千个 .md，慢且不相关。
SKIP_DIRS = {
    ".git", "node_modules", ".vitepress", "dist", "dist-check",
    "out", "cache", "__pycache__", ".venv", "venv", "data",
}


class PathArgsError(Exception):
    """入参有问题：路径不存在、后缀不对、目录里没有目标文件。"""


def _expand_dir(directory: str, exts: tuple[str, ...]) -> list[str]:
    """递归取目录下的目标文件，跳过 index.md 与依赖 / 产物目录。"""
    found = []
    for dirpath, dirnames, filenames in os.walk(directory):
        dirnames[:] = sorted(d for d in dirnames if d not in SKIP_DIRS)
        for fn in sorted(filenames):
            if fn in SKIP_NAMES:
                continue
            if any(fn.endswith(ext) for ext in exts):
                found.append(os.path.join(dirpath, fn))
    return sorted(found)


def collect(args: list[str] | None, default: str | None = DEFAULT_DOCS,
            exts: tuple[str, ...] = (".md",)) -> list[str]:
    """把命令行参数展开成待检查的文件列表。

    args 为空时退到 default；default 为 None 表示「必须显式给路径」。
    exts 是允许的后缀，目录递归按它取，单文件的中缀不在其中则报错。
    出错抛 PathArgsError，调用方负责打印与退出码。
    """
    args = [a for a in (args or []) if a and not a.startswith("-")]

    if not args:
        if default is None:
            raise PathArgsError("没有给路径：至少传一个文件或目录，例如 site/docs")
        if not os.path.isdir(default):
            raise PathArgsError(f"默认目录 {default} 不存在（当前工作目录：{os.getcwd()}）")
        return _expand_dir(default, exts)

    files: list[str] = []
    seen: set[str] = set()

    for raw in args:
        path = os.path.normpath(raw)

        if os.path.isdir(path):
            found = _expand_dir(path, exts)
            if not found:
                raise PathArgsError(f"目录 {raw} 下没有 {'/'.join(exts)} 文件（index.md 不计）")
            candidates = found
        elif os.path.isfile(path):
            if not path.endswith(exts):
                raise PathArgsError(f"{raw} 后缀不在允许范围（{' '.join(exts)}）")
            candidates = [path]
        elif "*" in raw or "?" in raw:
            candidates = [p for p in sorted(glob.glob(path, recursive=True))
                          if os.path.isfile(p) and p.endswith(exts)]
            if not candidates:
                raise PathArgsError(f"通配符 {raw} 没匹配到 {'/'.join(exts)} 文件")
        else:
            raise PathArgsError(f"路径不存在：{raw}")

        for path in candidates:
            key = os.path.abspath(path)
            if key in seen:
                continue
            seen.add(key)
            files.append(path)

    return files


def collect_many(paths: list[str], exts: tuple[str, ...] = (".md",)) -> list[str]:
    """多根目录集合：不存在的路径直接跳过（默认扫描用，不影响显式参数的严格校验）。"""
    out: list[str] = []
    seen: set[str] = set()
    for p in paths:
        if os.path.isdir(p):
            found = _expand_dir(p, exts)
        elif os.path.isfile(p):
            found = [p]
        else:
            continue
        for f in found:
            key = os.path.abspath(f)
            if key not in seen:
                seen.add(key)
                out.append(f)
    return out


def collect_or_exit(args: list[str] | None, default: str | None = DEFAULT_DOCS,
                    prog: str = "脚本", exts: tuple[str, ...] = (".md",)) -> list[str]:
    """collect 的 CLI 版本：出错打印中文提示并 exit(2)。"""
    try:
        return collect(args, default, exts)
    except PathArgsError as e:
        print(f"{prog}：{e}", file=sys.stderr)
        print(f"用法：python3 {prog} [文件或目录 ...]", file=sys.stderr)
        sys.exit(2)
