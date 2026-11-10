#!/usr/bin/env python3
"""全站链接体检。

用法：
    python3 scripts/check-links.py              # 查站内（路由、相对路径、锚点）
    python3 scripts/check-links.py --external   # additionally 跑外链 HEAD 检查
    python3 scripts/check-links.py --files      # 只列文件级汇总
    python3 scripts/check-links.py path/to/x.mdx

退出码：0 没有问题；1 有断链或死锚点。

为什么要单独一个脚本：链接是这类文档站最先腐坏的东西。改目录名、改课时标题、
拆模块，都会让链接悄悄失效，而构建本身不报错（Astro 对相对链接不做校验）。
CI 里跑这个，断链在 PR 阶段就红，而不是等读者点到 404。

判定口径：
    外部链接      http(s):// 开头。默认不查（慢、会因对方站点抖动而误报），--external 才查
    站内路由      / 开头。映射到 site/src/content/docs 下的文件
    相对路径      相对当前文件解析，必须存在于仓库
    锚点          #slug。用 github-slugger 的规则近似还原标题 slug
"""

import argparse
import os
import re
import sys
import unicodedata
import urllib.error
import urllib.request

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
DOCS = os.path.join(ROOT, "site/src/content/docs")
BASE = "/aidc-observability"  # astro.config.mjs 里的 base

# 扫哪些文件：站内课程、lab 说明、项目文档、根 README
PATTERNS = (
    ("site/src/content/docs", ".mdx"),
    ("labs", "README.md"),
    ("docs", ".md"),
    ("", "README.md"),
    ("", "README.en.md"),
)

LINK_RE = re.compile(r"(?<!!)\[([^\]]{0,200})\]\(([^)\s]+)(?:\s+\"[^\"]*\")?\)")
HEADING_RE = re.compile(r"^(#{1,6})\s+(.+?)\s*$", re.M)
FENCE_RE = re.compile(r"```.*?```|~~~.*?~~~", re.S)
INLINE_CODE_RE = re.compile(r"`[^`\n]*`")


def rel(path: str) -> str:
    r = os.path.relpath(path, ROOT)
    return path if r.startswith("..") else r


def targets() -> list:
    out = []
    for base, suffix in PATTERNS:
        d = os.path.join(ROOT, base) if base else ROOT
        if not os.path.isdir(d):
            continue
        for dirpath, dirnames, filenames in os.walk(d):
            dirnames[:] = [x for x in dirnames if x not in {".git", "node_modules", "dist", ".astro", "out"}]
            for fn in filenames:
                if fn.endswith(suffix) and (suffix != ".md" or base == "docs"):
                    out.append(os.path.join(dirpath, fn))
    return sorted(set(out))


def strip_code(text: str) -> str:
    """把代码块与行内代码换成等长空白，保持行号不变。"""
    def blank(m):
        return re.sub(r"[^\n]", " ", m.group(0))
    text = FENCE_RE.sub(blank, text)
    return INLINE_CODE_RE.sub(blank, text)


def slugify(heading: str) -> str:
    """近似 github-slugger：转小写、去标点、空格变连字符、保留中文。

    Starlight 用 github-slugger 生成锚点，规则与这里一致到「够用」为止：
    去掉的是一组固定 ASCII 标点，中文与数字原样保留。发现误报时先在这里加字符。
    """
    s = heading.strip().lower()
    s = re.sub(r"!\[[^\]]*\]\([^)]*\)", "", s)          # 图片
    s = re.sub(r"\[([^\]]*)\]\([^)]*\)", r"\1", s)      # 链接只留文字
    s = re.sub(r"[`*_~]", "", s)
    s = "".join(ch for ch in s if not unicodedata.category(ch).startswith("P") or ch in "-_")
    s = re.sub(r"\s+", "-", s)
    return s.strip("-")


def anchors_of(path: str) -> set:
    try:
        with open(path, encoding="utf-8") as f:
            text = strip_code(f.read())
    except OSError:
        return set()
    # frontmatter 里的 title 也会成为页面锚点（Starlight 的 h1 就是 title）
    out = set()
    fm = re.match(r"^---\n(.*?)\n---\n", text, re.S)
    if fm:
        m = re.search(r"^title:\s*(.+)$", fm.group(1), re.M)
        if m:
            out.add(slugify(m.group(1).strip().strip("\"'")))
    seen = {}
    for _, h in HEADING_RE.findall(text):
        slug = slugify(h)
        if not slug:
            continue
        # 同名标题：github-slugger 加 -1、-2 后缀
        n = seen.get(slug, 0)
        seen[slug] = n + 1
        out.add(slug if n == 0 else f"{slug}-{n}")
    return out


def route_to_path(route: str) -> str:
    """把站内路由映射成文件路径，找不到返回空串。"""
    clean = route.split("#", 1)[0].split("?", 1)[0]
    if clean.startswith(BASE):
        clean = clean[len(BASE):]
    clean = clean.strip("/")
    if clean == "":
        return os.path.join(DOCS, "index.mdx") if os.path.exists(os.path.join(DOCS, "index.mdx")) else ""
    for cand in (
        os.path.join(DOCS, clean + ".mdx"),
        os.path.join(DOCS, clean + ".md"),
        os.path.join(DOCS, clean, "index.mdx"),
    ):
        if os.path.exists(cand):
            return cand
    return ""


def resolve_relative(cur_file: str, route: str) -> str:
    """把相对链接解析成仓库内的文件路径。

    站内课程互相引用时写的是目录形式（`../02-lossless/05-pfc/`），
    那也是 Starlight 的真实路由，所以要按 .mdx / index.mdx 补齐再判。
    """
    base = os.path.normpath(os.path.join(os.path.dirname(cur_file), route.rstrip("/")))
    for cand in (base, base + ".mdx", base + ".md",
                 os.path.join(base, "index.mdx"), os.path.join(base, "index.md"),
                 os.path.join(base, "README.md")):
        if os.path.isfile(cand):
            return cand
    return ""


UA = {"User-Agent": "Mozilla/5.0 (compatible; aidc-observability-linkcheck/1.0)"}


def check_external(url: str, timeout: float = 8.0):
    """返回 ("", "") 表示通过，否则 (级别, 原因)。

    级别只有两种：
        warn   对方拒绝或无响应。403 是反爬，RemoteDisconnected 是网络抖动。
               这类不能当断链处理，否则每次跑都有几十条噪声
        error  确实不存在：404、410、域名解析失败
    """
    try:
        req = urllib.request.Request(url, method="HEAD", headers=UA)
        with urllib.request.urlopen(req, timeout=timeout):
            return "", ""
    except urllib.error.HTTPError as e:
        if e.code in (403, 405, 429):
            try:
                with urllib.request.urlopen(urllib.request.Request(url, headers=UA), timeout=timeout):
                    return "", ""
            except urllib.error.HTTPError as e2:
                if e2.code in (404, 410):
                    return "error", f"HTTP {e2.code}"
                return "warn", f"HTTP {e2.code}（对方拒绝访问）"
            except Exception:  # noqa: BLE001
                return "warn", f"HTTP {e.code}（对方拒绝访问）"
        if e.code in (404, 410):
            return "error", f"HTTP {e.code}"
        return "warn", f"HTTP {e.code}"
    except urllib.error.URLError as e:
        if "Name or service not known" in str(e.reason) or "nodename" in str(e.reason):
            return "error", "域名解析失败"
        return "warn", f"连接失败：{e.reason}"
    except Exception as e:  # noqa: BLE001
        return "warn", f"连接失败：{type(e).__name__}"


def check_file(path: str, external: bool):
    """检查单个文件，返回 (错误, 警告)，各为 [(行号, 链接, 原因)]。"""
    with open(path, encoding="utf-8") as f:
        raw = f.read()
    text = strip_code(raw)

    # 行号靠偏移量换算，代码块里被抹掉的链接不会误报
    line_starts = [0]
    for i, ch in enumerate(text):
        if ch == "\n":
            line_starts.append(i + 1)

    def line_of(off: int) -> int:
        lo, hi = 0, len(line_starts) - 1
        while lo < hi:
            mid = (lo + hi + 1) // 2
            if line_starts[mid] <= off:
                lo = mid
            else:
                hi = mid - 1
        return lo + 1

    problems, warnings = [], []
    for m in LINK_RE.finditer(text):
        target = m.group(2).strip()
        if not target or target.startswith(("mailto:", "tel:", "javascript:")):
            continue

        if target.startswith(("http://", "https://")):
            if external:
                level, why = check_external(target)
                if level == "error":
                    problems.append((line_of(m.start()), target, why))
                elif level == "warn":
                    warnings.append((line_of(m.start()), target, why))
            continue

        if target.startswith("#"):
            anchor = target[1:]
            if anchor and anchor not in anchors_of(path):
                problems.append((line_of(m.start()), target, "页内锚点不存在"))
            continue

        route, _, anchor = target.partition("#")

        if route.startswith("/"):
            dest = route_to_path(route)
            if not dest:
                problems.append((line_of(m.start()), target, "站内路由找不到对应文件"))
                continue
        else:
            dest = resolve_relative(path, route)
            if not dest:
                problems.append((line_of(m.start()), target, "相对路径不存在"))
                continue

        if anchor and dest.endswith((".mdx", ".md")):
            if anchor not in anchors_of(dest):
                problems.append((line_of(m.start()), target, f"目标文件里没有锚点 {anchor}"))
    return problems, warnings


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("paths", nargs="*", help="只查指定文件")
    ap.add_argument("--external", action="store_true", help="同时检查外链（慢）")
    ap.add_argument("--files", action="store_true", help="只列文件级汇总")
    ap.add_argument("--verbose", action="store_true", help="连外链警告也逐条列出")
    args = ap.parse_args()

    files = args.paths or targets()
    if not files:
        print("没找到要检查的文件", file=sys.stderr)
        return 1

    total_links = 0
    bad_files = 0
    all_problems = []
    all_warnings = []
    for f in files:
        probs, warns = check_file(f, args.external)
        if probs:
            bad_files += 1
            all_problems.extend((rel(f),) + p for p in probs)
        all_warnings.extend((rel(f),) + w for w in warns)
        with open(f, encoding="utf-8") as fh:
            total_links += len(LINK_RE.findall(strip_code(fh.read())))

    mode = "含外链" if args.external else "仅站内"
    print(f"链接体检（{mode}）· {len(files)} 个文件 · {total_links} 条链接")
    print()

    if all_warnings:
        print(f"⚠ {len(all_warnings)} 条外链没验成（对方拒绝或无响应，不当断链处理）")
        if args.verbose:
            for f, line, target, why in all_warnings:
                print(f"  {f}:{line}  {target}  {why}")
        print()

    if not all_problems:
        print("✅ 全部链接可达")
        return 0

    if not args.files:
        for f, line, target, why in all_problems:
            print(f"  {f}:{line}  {target}")
            print(f"      {why}")
        print()

    by_file = {}
    for f, *_ in all_problems:
        by_file[f] = by_file.get(f, 0) + 1
    for f, n in sorted(by_file.items(), key=lambda kv: -kv[1]):
        print(f"  {n:3d} 处  {f}")
    print()
    print(f"❌ {len(all_problems)} 处问题，涉及 {bad_files} 个文件")
    return 1


if __name__ == "__main__":
    sys.exit(main())
