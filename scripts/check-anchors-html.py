#!/usr/bin/env python3
"""构建产物爬虫：在 VitePress 生成的 HTML 上验活锚点与站内路由。

用法：
    cd site && pnpm build && cd ..
    python3 scripts/check-anchors-html.py                 # 默认扫 site/.vitepress/dist
    python3 scripts/check-anchors-html.py --dist site/.vitepress/dist --verbose

退出码：0 通过；1 有死锚点或死路由；2 找不到构建产物。

与 check-links.py 的分工：
    check-links.py       读 Markdown 源码，按 markdown-it-anchor 的规则**近似**还原
                         slug。它能在 PR 阶段就红，但两次近似（slug 规则、标题解析）
                         都可能与真实渲染结果差一点。
    check-anchors-html.py 直接读构建后的 HTML，`id="..."` 与 `href="..."` 都是
                         渲染器写完的成品。slug 规则变了，这里会第一时间发现。

只查站内链接。外链不碰：CI 里抓外链会被反爬与网络抖动搞成噪声。
"""

from __future__ import annotations

import argparse
import html
import os
import re
import sys
from collections import defaultdict

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
DEFAULT_DIST = os.path.join(ROOT, "site", ".vitepress", "dist")
# .vitepress/config.mts 里的 base，末尾不带的写法统一在这里补
BASE = "/aidc-observability"

ID_RE = re.compile(r'\sid="([^"]+)"')
HREF_RE = re.compile(r'\shref="([^"]+)"')


def to_local(path: str) -> str | None:
    """把站点 URL 路径映射到 dist 里的 HTML 文件；映射不上返回 None。"""
    if not path.startswith(BASE + "/") and path != BASE and path != BASE + "/":
        return None

    rel = path[len(BASE):].lstrip("/")
    rel = rel.split("?", 1)[0]
    if not rel:
        rel = "index.html"
    elif rel.endswith("/"):
        rel = rel + "index.html"
    elif not os.path.splitext(rel)[1]:
        # cleanUrls: true → /foo/bar 落成 foo/bar.html
        rel = rel + ".html"

    full = os.path.join(DIST, rel)
    return full if os.path.isfile(full) else None


def page_ids(path: str) -> set:
    try:
        text = open(path, encoding="utf-8").read()
    except OSError:
        return set()
    return {html.unescape(x) for x in ID_RE.findall(text)}


def scan(verbose: bool = False) -> int:
    if not os.path.isdir(DIST):
        print(f"找不到构建产物：{DIST}\n先跑：cd site && pnpm build", file=sys.stderr)
        return 2

    pages = []
    for dirpath, dirnames, filenames in os.walk(DIST):
        dirnames[:] = [d for d in dirnames if d not in {"assets", "hashmap.json", ".vite"}]
        for fn in filenames:
            if fn.endswith(".html"):
                pages.append(os.path.join(dirpath, fn))
    pages.sort()

    if not pages:
        print(f"{DIST} 下没有 HTML，构建可能失败了", file=sys.stderr)
        return 2

    id_sets = {}
    links = defaultdict(list)   # 源页 → [(href, 行号近似)]
    for page in pages:
        text = open(page, encoding="utf-8").read()
        id_sets[page] = {html.unescape(x) for x in ID_RE.findall(text)}
        for m in HREF_RE.finditer(text):
            links[page].append(m.group(1))

    dead_anchor, dead_route = [], []
    total_internal = 0
    for page, hrefs in links.items():
        for href in hrefs:
            href = html.unescape(href)
            if href.startswith(("http://", "https://", "mailto:", "javascript:", "tel:")):
                continue
            if not href.startswith("/"):
                # 相对链接（同页锚点 #xxx 除外）在 VitePress 产物里基本不出现，
                # 出现就说明是外挂脚本或手动插入的 HTML，跳过不误报
                if href.startswith("#"):
                    frag = href[1:]
                    if frag and frag not in id_sets.get(page, set()):
                        dead_anchor.append((page, href, "同页锚点不存在"))
                continue

            total_internal += 1
            route, _, frag = href.partition("#")
            dest = to_local(route)
            if dest is None:
                dead_route.append((page, href, "站内路由没有对应 HTML"))
                continue
            if frag and frag not in id_sets.get(dest, set()):
                dead_anchor.append((page, href, f"{os.path.relpath(dest, DIST)} 里没有 id={frag}"))

    def show(items, title):
        by_page = defaultdict(list)
        for page, href, why in items:
            by_page[page].append((href, why))
        print(f"\n❌ {title}：{len(items)} 处，涉及 {len(by_page)} 个页面")
        for page, rows in sorted(by_page.items(), key=lambda kv: -len(kv[1])):
            rel = os.path.relpath(page, DIST)
            print(f"   {rel}  ({len(rows)} 处)")
            if verbose:
                for href, why in rows[:20]:
                    print(f"      {href}    {why}")

    if not (dead_anchor or dead_route):
        print(f"✅ 构建产物锚点全活：{len(pages)} 个页面 · {total_internal} 条站内链接")
        return 0

    if dead_anchor:
        show(dead_anchor, "死锚点")
    if dead_route:
        show(dead_route, "死路由")
    print("\n加 --verbose 看逐条明细。")
    return 1


def main() -> int:
    global DIST
    ap = argparse.ArgumentParser()
    ap.add_argument("--dist", default=DEFAULT_DIST, help="构建产物目录")
    ap.add_argument("--verbose", action="store_true", help="逐条列出问题链接")
    args = ap.parse_args()
    DIST = os.path.abspath(args.dist)
    return scan(args.verbose)


DIST = DEFAULT_DIST

if __name__ == "__main__":
    sys.exit(main())
