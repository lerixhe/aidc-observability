#!/usr/bin/env python3
"""原型 HTML → VitePress Markdown 骨架转换器。

定位：只做**机械转换**（约 70%），语义改写、脱敏、补参考资料必须人工完成。
转换完成后脚本会打印一份「待人工处理清单」。

映射规则：
    header.hero              -> frontmatter title / description
    div.breadcrumb / div.nav -> 丢弃（VitePress 自带导航与上一课/下一课分页）
    div.card + h2            -> ## 小节
    div.compare（old/new）   -> Markdown 表格
    div.link-exp             -> ::: info 经验映射
    div.warn                 -> ::: warning
    div.goal                 -> ::: tip 本节目标
    div.keycard              -> ::: tip 本课要点
    .quiz-item + <details>   -> ::: details 参考答案与评分维度
    <style> / <footer>       -> 丢弃

用法：
    python3 scripts/html2mdx.py reference/archive-aidc/lesson-01.html -o /tmp/l1.md
    python3 scripts/html2mdx.py reference/archive-aidc/lesson-03.html -o ... --title "RoCEv2：把 RDMA 搬到以太网"

注意：原型目录含个人化叙述与前雇主信息（.gitignore 已排除），
产出物在人工改写（去个人化、脱敏）前**不得入库**。
"""

from __future__ import annotations

import argparse
import re
import sys
from html.parser import HTMLParser
from pathlib import Path

VOID_TAGS = {"br", "hr", "img", "meta", "link", "input", "source"}
SKIP_TAGS = {"style", "script", "footer", "head", "title"}

# 需要人工补齐的固定结构（每课统一结构）
REQUIRED_SECTIONS = [
    "本节目标",
    "机制拆解",
    "经验映射",
    "判读与实操",
    "练习",
    "本课要点",
    "参考资料",
]


class Node:
    """极简 DOM 节点。children 里同时容纳 str 与 Node。"""

    def __init__(self, tag: str, attrs: dict[str, str] | None = None):
        self.tag = tag
        self.attrs = attrs or {}
        self.children: list = []

    def __repr__(self) -> str:  # 便于调试
        return f"<{self.tag} class={self.attrs.get('class')!r}>"

    @property
    def classes(self) -> list[str]:
        return self.attrs.get("class", "").split()

    def has_class(self, name: str) -> bool:
        return name in self.classes

    def iter(self):
        """深度优先遍历所有元素节点。"""
        for child in self.children:
            if isinstance(child, Node):
                yield child
                yield from child.iter()

    def find(self, *, tag: str | None = None, cls: str | None = None) -> list[Node]:
        out = []
        for node in self.iter():
            if tag and node.tag != tag:
                continue
            if cls and not node.has_class(cls):
                continue
            out.append(node)
        return out

    def first(self, *, tag: str | None = None, cls: str | None = None) -> Node | None:
        found = self.find(tag=tag, cls=cls)
        return found[0] if found else None


class TreeBuilder(HTMLParser):
    def __init__(self):
        super().__init__(convert_charrefs=True)
        self.root = Node("#root")
        self.stack = [self.root]

    def handle_starttag(self, tag, attrs):
        node = Node(tag, {k: (v or "") for k, v in attrs})
        self.stack[-1].children.append(node)
        if tag not in VOID_TAGS:
            self.stack.append(node)

    def handle_startendtag(self, tag, attrs):
        self.stack[-1].children.append(Node(tag, {k: (v or "") for k, v in attrs}))

    def handle_endtag(self, tag):
        for i in range(len(self.stack) - 1, 0, -1):
            if self.stack[i].tag == tag:
                del self.stack[i:]
                return

    def handle_data(self, data):
        self.stack[-1].children.append(data)


def parse_html(text: str) -> Node:
    builder = TreeBuilder()
    builder.feed(text)
    return builder.root


# ---------------------------------------------------------------- 行内渲染

INLINE_PREFIX = {"strong": "**", "b": "**", "em": "*", "i": "*"}


def render_inline(node) -> str:
    """把节点内容渲染成 Markdown 行内文本。"""
    if isinstance(node, str):
        # 压缩空白：HTML 里的换行与缩进不应进入正文
        return re.sub(r"\s+", " ", node)

    tag = node.tag
    if tag in SKIP_TAGS:
        return ""

    inner = "".join(render_inline(child) for child in node.children).strip()

    if tag in INLINE_PREFIX:
        mark = INLINE_PREFIX[tag]
        return f"{mark}{inner}{mark}" if inner else ""
    if tag == "code":
        return f"`{inner}`" if inner else ""
    if tag == "a":
        href = node.attrs.get("href", "")
        if not href or href.startswith("#"):
            return inner
        return f"[{inner}]({href})"
    if tag == "br":
        return "  \n"
    if tag == "span":
        # 练习类型标记（必答 / 自测）单独处理，此处只取文本
        return inner
    return inner


def plain_text(node) -> str:
    """取节点的纯文本（去掉所有标记），用于识别题号等。"""
    if isinstance(node, str):
        return node
    return "".join(plain_text(c) for c in node.children)


def normalize(text: str) -> str:
    """清理行内文本里多余的空白。"""
    return re.sub(r"[ \t]{2,}", " ", text).strip()


# ---------------------------------------------------------------- 块级渲染


def render_list(node: Node) -> str:
    ordered = node.tag == "ol"
    lines = []
    for i, item in enumerate(c for c in node.children if isinstance(c, Node) and c.tag == "li"):
        body = render_inline(item)
        prefix = f"{i + 1}. " if ordered else "- "
        lines.append(prefix + body)
    return "\n".join(lines)


def render_compare(node: Node) -> str:
    """div.compare 双栏 -> Markdown 表格（语义对齐需人工核对）。"""
    old = node.first(cls="old")
    new = node.first(cls="new")
    if not old or not new:
        return render_plain(node)

    def heading(box: Node) -> str:
        h = box.first(tag="h4") or box.first(tag="h3")
        return render_inline(h) if h else ""

    def items(box: Node) -> list[str]:
        li = box.first(tag="ul")
        if li:
            return [render_inline(x) for x in li.children if isinstance(x, Node) and x.tag == "li"]
        return [render_inline(p) for p in box.find(tag="p")]

    left, right = items(old), items(new)
    rows = []
    for i in range(max(len(left), len(right))):
        a = left[i] if i < len(left) else ""
        b = right[i] if i < len(right) else ""
        rows.append(f"| {a} | {b} |")

    table = "\n".join(
        [
            f"| {heading(old)} | {heading(new)} |",
            "| --- | --- |",
            *rows,
        ]
    )
    return "<!-- MIGRATION: 双栏对比已转成表格，行语义请人工核对 -->\n" + table


def render_details(node: Node) -> str:
    summary = node.first(tag="summary")
    if summary is not None:
        # 原 summary 文案（如「参考答案（先自己答，再展开）」）不再使用
        summary.children = []

    body_parts = [render_block(c) for c in node.children if not (isinstance(c, Node) and c.tag == "summary")]
    body = "\n\n".join(p for p in body_parts if p.strip())
    return f"::: details 参考答案与评分维度\n\n{body}\n\n:::"


def render_quiz_item(node: Node) -> str:
    """quiz-item -> 加粗题号 + 题干 + 折叠答案。"""
    qtype_node = node.first(cls="must") or node.first(cls="self")
    label = "必答" if qtype_node and qtype_node.has_class("must") else "自测"

    # 题干：去掉类型 chip 与 details 后的行内内容，并抽出题号（原型的 <b>Q1</b>）
    parts = []
    details_node = None
    qno = ""
    for child in node.children:
        if isinstance(child, Node):
            if child.tag == "details":
                details_node = child
                continue
            if child.tag == "span" and (child.has_class("must") or child.has_class("self")):
                continue
            if not qno and child.tag in ("b", "strong"):
                m = re.fullmatch(r"(Q\d+)", normalize(plain_text(child)))
                if m:
                    qno = m.group(1)
                    continue
            parts.append(render_inline(child))
        else:
            parts.append(re.sub(r"\s+", " ", child))

    question = normalize("".join(parts)).lstrip("：: ")
    if not qno:
        m = re.match(r"^(Q\d+)\s*[：:]\s*(.*)$", question)
        if m:
            qno, question = m.group(1), m.group(2)
    lead = f"**{label} {qno}**：{question}" if qno else f"**{label}**：{question}"
    out = [lead]
    if details_node is not None:
        out.append(render_details(details_node))
    return "\n\n".join(out)


CONTAINER = {"tip": "tip", "note": "info", "caution": "warning", "danger": "danger"}


def render_aside(node: Node, kind: str, title: str | None) -> str:
    body = render_plain(node)
    body = re.sub(r"^[⚠️❗️📌\s]+", "", body)
    head = f"::: {CONTAINER[kind]} {title}" if title else f"::: {CONTAINER[kind]}"
    return f"{head}\n\n{body}\n\n:::"


BLOCK_TAGS = {
    "p", "ul", "ol", "li", "h1", "h2", "h3", "h4", "pre", "table", "tr", "div",
    "details", "section", "article", "blockquote", "aside", "header", "footer",
}


def render_plain(node: Node) -> str:
    """通用块级渲染：段落 / 列表 / 标题 / 代码块。"""
    if not any(isinstance(c, Node) and c.tag in BLOCK_TAGS for c in node.children):
        # 纯行内内容（含 <b>/<code>/<a>）：一次性走行内渲染，不要按节点切块
        return normalize(render_inline(node))

    blocks = []
    for child in node.children:
        blocks.append(render_block(child))
    return "\n\n".join(b for b in blocks if b.strip())


def render_table(node: Node) -> str:
    rows = []
    for tr in node.find(tag="tr"):
        cells = [normalize(render_inline(c)) for c in tr.children if isinstance(c, Node) and c.tag in ("td", "th")]
        rows.append("| " + " | ".join(cells) + " |")
    if not rows:
        return ""
    cols = rows[0].count("|") - 1
    header = "| " + " | ".join(["---"] * cols) + " |"
    return "\n".join([rows[0], header] + rows[1:])


def render_pre(node: Node) -> str:
    code_node = node.first(tag="code")
    target = code_node if code_node is not None else node
    raw = "".join(c for c in target.children if isinstance(c, str))
    lang = ""
    for cls in target.classes:
        if cls.startswith("language-"):
            lang = cls.split("-", 1)[1]
    return f"```{lang}\n{raw.strip()}\n```"


def render_keycard(node: Node) -> str:
    parts = []
    for child in node.children:
        if isinstance(child, Node) and child.tag in ("h2", "h3"):
            continue  # 标题由容器 title 承担
        parts.append(render_block(child))
    body = "\n\n".join(p for p in parts if p.strip())
    return f"::: tip 本课要点\n\n{body}\n\n:::"


def render_block(node) -> str:
    if isinstance(node, str):
        text = normalize(node)
        return text

    tag = node.tag
    if tag in SKIP_TAGS or node.has_class("breadcrumb") or node.has_class("nav"):
        return ""

    if node.has_class("compare"):
        return render_compare(node)
    if node.has_class("link-exp"):
        return render_aside(node, "note", "经验映射（IDC → AIDC）")
    if node.has_class("warn"):
        return render_aside(node, "caution", None)
    if node.has_class("goal"):
        return render_aside(node, "tip", "本节目标")
    if node.has_class("keycard"):
        return render_keycard(node)
    if node.has_class("quiz-item"):
        return render_quiz_item(node)
    if tag == "details":
        return render_details(node)
    if tag == "table":
        return render_table(node)
    if tag == "pre":
        return render_pre(node)
    if tag == "p":
        return normalize(render_inline(node))
    if tag in ("ul", "ol"):
        return render_list(node)
    if tag in ("h1", "h2", "h3", "h4"):
        level = {"h1": "#", "h2": "##", "h3": "###", "h4": "####"}[tag]
        text = normalize(render_inline(node))
        # 去掉原型里的 emoji 前缀（站点侧由容器图标承担）
        text = re.sub(r"^[📝🎓🔗⚠️\s]+", "", text)
        return f"{level} {text}"

    return render_plain(node)


# ---------------------------------------------------------------- 主流程


def hero_info(root: Node) -> tuple[str, str]:
    hero = root.first(cls="hero") or root.first(tag="h1")
    if hero is None:
        return "未命名课程", ""
    h1 = hero.first(tag="h1") if hero.tag != "h1" else hero
    title = normalize(render_inline(h1)) if h1 is not None else "未命名课程"
    para = hero.first(tag="p")
    description = normalize(render_inline(para)) if para is not None else ""
    return title, description


def collect_sections(root: Node) -> list[str]:
    """按文档顺序收集正文块（跳过 hero / breadcrumb / nav）。"""
    blocks: list[str] = []

    def visit(node: Node):
        for child in node.children:
            if isinstance(child, str):
                text = normalize(child)
                if text:
                    blocks.append(text)
                continue
            if child.tag in SKIP_TAGS:
                continue
            if child.has_class("hero") or child.has_class("breadcrumb") or child.has_class("nav"):
                continue
            if child.has_class("wrap") or child.tag in ("body", "html", "#root"):
                visit(child)
                continue
            blocks.append(render_block(child))

    visit(root)
    return [b for b in blocks if b and b.strip()]


def build_markdown(title: str, description: str, blocks: list[str]) -> str:
    body = "\n\n".join(blocks)
    def yaml(value: str) -> str:
        return '"' + value.replace("\\", "\\\\").replace('"', '\\"') + '"'

    frontmatter = (
        "---\n"
        f"title: {yaml(title)}\n"
        f"description: {yaml(description)}\n"
        "---\n"
    )
    return frontmatter + "\n" + body + "\n"


def audit(md: str) -> list[str]:
    """检查每课统一结构的缺口，供人工补齐。"""
    todos = []
    for section in REQUIRED_SECTIONS:
        if not re.search(rf"^##\s+{re.escape(section)}", md, re.MULTILINE):
            todos.append(f"缺少小节：## {section}")
    if "发给 AI" in md or "发给AI" in md:
        todos.append("练习说明仍含「发给 AI 批改」，改为自答 + 自评")
    if re.search(r"^<!-- MIGRATION", md, re.MULTILINE):
        todos.append("存在 <!-- MIGRATION --> 标记，需人工核对")
    return todos


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="原型 HTML → VitePress Markdown 骨架")
    parser.add_argument("source", type=Path, help="原型 HTML 文件，如 reference/archive-aidc/lesson-01.html")
    parser.add_argument("-o", "--output", type=Path, help="输出 Markdown 路径（默认打印到 stdout）")
    parser.add_argument("--title", help="覆盖 frontmatter title（传课时标题的规范写法）")
    args = parser.parse_args(argv)

    if not args.source.is_file():
        print(f"找不到源文件：{args.source}", file=sys.stderr)
        return 1

    root = parse_html(args.source.read_text(encoding="utf-8"))
    title, description = hero_info(root)
    blocks = collect_sections(root)
    md = build_markdown(args.title or title, description, blocks)

    if args.output:
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(md, encoding="utf-8")

    todos = audit(md)
    print(f"源文件：{args.source}")
    print(f"标题：{args.title or title}")
    print(f"产出：{args.output or '(stdout)'}  行数：{md.count(chr(10))}")
    if todos:
        print("\n待人工处理：")
        for item in todos:
            print(f"  - {item}")
    if not args.output:
        print("\n" + "-" * 60 + "\n")
        print(md)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
