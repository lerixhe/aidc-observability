#!/usr/bin/env python3
"""课程文风体检：对照 corpus/style/COURSE-STYLE.md §2 的十条硬指标。

用法：
    python3 scripts/style-check.py                 # 汇总报表 + 逐课明细
    python3 scripts/style-check.py --files         # 只看每课一行
    python3 scripts/style-check.py path/to/xx.md   # 单文件
    python3 scripts/style-check.py site/docs/01-paradigm site/docs/02-lossless  # 多目录
    python3 scripts/style-check.py 'site/docs/**/*.md'   # 通配符

退出码：0 全部达标；1 有指标标红（CI 可用）；2 入参有误（路径不存在等）。

入参规则统一在 scripts/mdpaths.py，与 long-sents.py 共用一份。

统计范围：已剔除 frontmatter、代码块、表格、import、参考资料 —— 只量正文。
分句口径：逗号与句末之间的一段（同 COURSE-STYLE §4 规则一），顿号、冒号不切分。
"""

import re
import statistics as st
import sys

import mdpaths

DOCS = "site/docs"

# --- 目标值（COURSE-STYLE §2 / §2.1 / §7）---
# 口径：只统计正文的 6 类句读符号（，。、；：？！），已剔除 frontmatter / 代码块 / 表格
#
# 分两档，依据 COURSE-STYLE §2.1：
#   硬项  不达标就是返工项（分号 0、感叹号 0、超长句、句长区间、标点配额底线）
#   观察项 与硬项在几何上互相牵制，达标靠改写已无收益。照实打印，不阻塞 CI
#
# §2.1 的原话：让逗号占比高就要少用句号，句子就更长 → P90 上升；
# 让 P90 达标就要多断句 → 逗号占比下降。两者不能同时最大化，所以分档。
HARD = {
    "clause_mean": (12, 16),      # 分句均长
    "clause_median": (11, 14),    # 分句中位
    "sent_over60": (0, 3),        # 超 60 字句占比 %
    "semicolon": (0, 0.5),        # 分号占比 %
    "comma_dom": (30, 100),       # 逗号占比 %（§2.1 放宽后底线）
    "colon": (0, 20),             # 冒号占比 %
    "dunhao": (8, 100),           # 顿号占比 %
    "bang": (0, 0),               # 感叹号占比 %
}
SOFT = {
    "clause_short10": (45, 100),  # ≤10 字分句占比 %，理想值
    "sent_p90": (0, 40),          # 整句 P90，理想值
}
TARGET = {**HARD, **SOFT, "trans_word": (3, 10**9)}  # trans_word 按课判，不进汇总

BANNED = {
    "不是A而是B": r"不是[^。]{1,20}而是",
    "为了…，": r"为了[^。]{0,15}，",
    "当…时": r"当[^。，]{1,12}时[，,]",
    "事实上/实际上": r"事实上|实际上",
    "值得注意的是族": r"值得注意的是|需要注意的是|值得一提的是",
    "随着…的发展": r"随着.{0,12}的(发展|普及|演进)",
    "在当今/瞬息万变": r"在当今|瞬息万变",
    "从…的角度来看": r"从[^。]{1,12}的角度(来看|看)",
    "让我们": r"让我们",
    "在这个过程中": r"在这个过程中",
    "综上所述/总而言之": r"综上所述|总而言之",
    "之所以…是因为": r"之所以[^。]{0,20}是因为",
    "看似…实则": r"看似[^。]{1,15}实则",
    "与其说…不如说": r"与其说[^。]{0,20}不如说",
    "首先…其次": r"首先.{0,80}?其次",
    "有一种A就是B": r"有一种[^。]{1,12}，?就是",
    "不仅…而且": r"不仅[^。]{0,20}(而且|更)",
    "无论是…还是": r"无论是[^。]{0,20}还是",
    "如果说…那么": r"如果说[^。]{0,20}那么",
    "直到…才": r"直到[^。]{0,20}才",
}
TRANS = ("也就是", "相当于", "等价于", "类比", "本质上")
PUNCT = "，。、；：？！"


def clean(text: str) -> str:
    text = re.sub(r"^---.*?\n---\n", "", text, flags=re.S)     # frontmatter
    text = re.sub(r"```.*?```", "", text, flags=re.S)          # 代码块
    text = re.sub(r"^\s*\|.*$", "", text, flags=re.M)          # 表格
    text = re.sub(r"^\s*import .*$", "", text, flags=re.M)      # import
    text = re.sub(r"\n## 参考资料.*?(?=\n## |\Z)", "\n", text, flags=re.S)  # 书目不算正文
    return text


def strip_md(s: str) -> str:
    # 链接只留文字、不留 URL：参考资料行的 URL 不算句子成分（COURSE-STYLE §2 口径）
    s = re.sub(r"\[([^\]]*)\]\([^)]*\)", r"\1", s)
    s = re.sub(r"https?://\S+", "", s)
    return re.sub(r"[*#>\-\s`]", "", s)


def measure(text: str) -> dict:
    body = clean(text)
    m = {}
    for c in PUNCT:
        m[c] = body.count(c)
    total = sum(m.values()) or 1

    sents, clauses = [], []
    for s in re.split(r"[。！？\n]", body):
        s = strip_md(s)
        if len(s) > 3:
            sents.append(len(s))
    # 分句口径按 COURSE-STYLE §4：逗号（或句末）之间的一段；顿号、冒号是段内元素，不切分
    for c in re.split(r"[，。；？！\n]", body):
        c = strip_md(c)
        if len(c) > 1:
            clauses.append(len(c))

    m["sents"] = sents
    m["clauses"] = clauses
    m["clause_mean"] = sum(clauses) / len(clauses) if clauses else 0
    m["clause_median"] = st.median(clauses) if clauses else 0
    m["clause_short10"] = 100 * sum(1 for x in clauses if x <= 10) / len(clauses) if clauses else 0
    m["sent_mean"] = sum(sents) / len(sents) if sents else 0
    m["sent_median"] = st.median(sents) if sents else 0
    m["sent_p90"] = sorted(sents)[int(len(sents) * 0.9)] if sents else 0
    m["sent_max"] = max(sents) if sents else 0
    m["sent_over60"] = 100 * sum(1 for x in sents if x > 60) / len(sents) if sents else 0
    m["semicolon"] = 100 * m["；"] / total
    m["comma_dom"] = 100 * m["，"] / total
    m["dunhao"] = 100 * m["、"] / total
    m["question"] = 100 * m["？"] / total
    m["bang"] = 100 * m["！"] / total
    m["trans_word"] = sum(body.count(w) for w in TRANS)
    m["banned"] = {k: len(re.findall(p, body)) for k, p in BANNED.items()}
    m["banned"] = {k: v for k, v in m["banned"].items() if v}
    m["ni"] = len(re.findall(r"你(?!们)", body))
    m["wo"] = len(re.findall(r"我(?!们)", body))
    m["women"] = body.count("我们")

    # 问号分层：标题类（## / Aside title）/ 练习题 / 正文反问
    head_q, quiz_q, body_q = 0, 0, 0
    for line in body.splitlines():
        n = line.count("？")
        if not n:
            continue
        if re.match(r"\s*#{2,6} ", line) or re.search(r"<Aside[^>]*title=", line):
            head_q += n
        elif re.search(r"必答|Q\d|<summary|自测|练习|参考答案", line):
            quiz_q += n
        else:
            body_q += n
    m["ask_head"], m["ask_quiz"], m["ask_body"] = head_q, quiz_q, body_q
    return m


def ok(key: str, val: float) -> bool:
    lo, hi = TARGET[key]
    return lo <= val <= hi


def flags_of(m: dict) -> list[str]:
    """单课的观察标记。这些不进汇总红项，只提示哪里值得看一眼。"""
    flags = []
    if m["semicolon"] > 0.5:
        flags.append(f"；{m['；']}个")
    if m["sent_over60"] > 3:
        flags.append(f"长句{m['sent_over60']:.0f}%")
    if m["trans_word"] < 3:
        flags.append(f"译词{m['trans_word']}")
    if m["banned"]:
        flags.append("/".join(f"{k}×{v}" for k, v in m["banned"].items()))
    if m["bang"]:
        flags.append(f"感叹号{m['！']}")
    if m["ask_head"]:
        flags.append(f"问句标题{m['ask_head']}")
    if m["ask_body"] > 1:
        flags.append(f"正文反问{m['ask_body']}")
    if m["ni"] < m["wo"] * 3:
        flags.append(f"人称你/我={m['ni']}/{m['wo']}")
    # 分句均长按课看：单课超 16 不阻塞（汇总达标），但值得记一笔
    if m["clause_mean"] > 16:
        flags.append(f"分句偏长{m['clause_mean']:.1f}")
    return flags


def blocking_flags(m: dict) -> list[str]:
    """单课里真正踩到硬项的那几条：分号、禁用句、感叹号、问句标题。

    `--files` 模式只看这些：一句禁用句不应该被汇总平均掉。
    """
    out = []
    if m["semicolon"] > 0.5:
        out.append("分号")
    if m["bang"]:
        out.append("感叹号")
    if m["banned"]:
        out.append("禁用句")
    if m["ask_head"]:
        out.append("问句标题")
    return out


def main() -> int:
    argv = sys.argv[1:]
    only_files = "--files" in argv
    files = mdpaths.collect_or_exit([a for a in argv if not a.startswith("-")], DOCS,
                                    prog="style-check.py")
    per_file, red = {}, []

    for f in files:
        try:
            m = measure(open(f, encoding="utf8").read())
        except OSError as e:
            print(f"style-check.py：读不了 {f}：{e}", file=sys.stderr)
            return 2
        if not m["sents"]:
            print(f"style-check.py：{f} 没有可统计的正文（全是代码块或表格？）", file=sys.stderr)
            return 2
        per_file[f] = m

    # 汇总
    allc = [x for m in per_file.values() for x in m["clauses"]]
    alls = [x for m in per_file.values() for x in m["sents"]]
    tot = {c: sum(m[c] for m in per_file.values()) for c in PUNCT}
    tsum = sum(tot.values()) or 1
    agg = {
        "clause_mean": sum(allc) / len(allc), "clause_median": st.median(allc),
        "clause_short10": 100 * sum(1 for x in allc if x <= 10) / len(allc),
        "sent_p90": sorted(alls)[int(len(alls) * 0.9)],
        "sent_over60": 100 * sum(1 for x in alls if x > 60) / len(alls),
        "semicolon": 100 * tot["；"] / tsum, "comma_dom": 100 * tot["，"] / tsum,
        "colon": 100 * tot["："] / tsum,
        "dunhao": 100 * tot["、"] / tsum, "bang": 100 * tot["！"] / tsum,
    }

    print(f"课程文风体检 · {len(files)} 个文件 · 正文分句 {len(allc)} / 句 {len(alls)}\n")

    if only_files:
        # --files：只出每课一行，供 CI 日志快速过一眼；红项判定仍然生效
        for f, m in per_file.items():
            flags = flags_of(m)
            name = f.split("/")[-1]
            print(f"{name:<22}句均{m['sent_mean']:>5.1f} 分句{m['clause_mean']:>4.1f} "
                  f"{'⚠ ' + ' '.join(flags) if flags else 'OK'}")
            if blocking_flags(m):
                red.append(f)
        print(f"\n汇总红项：{len(red)} 项 {red if red else ''}")
        if red:
            print("改法见 COURSE-STYLE §10。")
        return 1 if red else 0
    print(f"{'硬项（不达标即返工）':<18}{'实测':>8}{'目标':>12}  判定")
    print("-" * 46)
    for k, (lo, hi) in HARD.items():
        v = agg[k]
        good = ok(k, v)
        rng = f"{lo}–{hi}" if hi < 10**8 else f"≥{lo}"
        print(f"{k:<18}{v:>8.1f}{rng:>12}  {'✅' if good else '❌'}")
        if not good:
            red.append(k)

    print(f"\n{'观察项（§2.1 已论证不可兼得）':<18}{'实测':>8}{'目标':>12}  判定")
    print("-" * 46)
    for k, (lo, hi) in SOFT.items():
        v = agg[k]
        rng = f"{lo}–{hi}" if hi < 10**8 else f"≥{lo}"
        print(f"{k:<18}{v:>8.1f}{rng:>12}  {'✅' if ok(k, v) else '⚠ 放行'}")

    # 逐课红项
    print("\n逐课明细（分号 / 超60字% / 翻译词 / 禁用句）")
    print("-" * 72)
    for f, m in per_file.items():
        flags = flags_of(m)
        name = f.split("/")[-1]
        print(f"{name:<22}句均{m['sent_mean']:>5.1f} 分句{m['clause_mean']:>4.1f} "
              f"{'⚠ ' + ' '.join(flags) if flags else 'OK'}")

    print(f"\n汇总红项：{len(red)} 项 {red if red else ''}")
    if red:
        print("改法见 COURSE-STYLE §10；观察项不在红项里，别再为它改写正文。")
    return 1 if red else 0


if __name__ == "__main__":
    sys.exit(main())
