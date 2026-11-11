# 语料库（corpus）

> 收集作者本人的口癖、语言习惯、文字风格。

与 `reference/` 的区别：`reference/archive-aidc/` 是**内容底本**（讲什么），
这里是**语体底本**（怎么讲）。

---

## 目录结构

```
corpus/
├── README.md                      本文件
└── sources/                       原始语料存档（原文保真，逐篇一份，不入库）
    └── 2026-02-27-api-to-ui.md    《从API到UI：AI助力后端工程师全栈交付》
```

## 规范体系已废除

v2.1 起（D21），这里的全部文风规范与检查工具一并删除：

| 删掉的 | 原来干什么 |
|---|---|
| `style/COURSE-STYLE.md` | 十条硬指标、禁用清单、章节骨架、六型开头 |
| `style/STYLE.md`、`style/LEXICON.md`、`style/NARRATIVE-DIGEST-tcp-book.md` | 特征卡、口癖词表、小册手法拆解 |
| `scripts/style-check.py`、`scripts/long-sents.py` | 量化体检与超长句清单 |
| `scripts/check-titles.py` | 标题制式校验 |
| `scripts/normalize-quotes.py` | 引号口径统一 |

写作不再参照本目录的任何文件。写完自己读一遍，哪里拗口改哪里，就是全部标准。

## 语料存档

`sources/` 只做原文存档，供以后回查，不再产出提炼文档。

- 命名：`YYYY-MM-DD-<slug>.md`，保留原样不编辑（错字也是风格证据）
- 含内部平台名、厂商痕迹、内部链接的原文已在 `.gitignore` 排除，仅本地保留
- 引用例句前先脱敏：平台名泛化、链接删除、规模数字改量级
