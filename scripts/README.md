# scripts/

| 脚本 | 状态 | 用途 |
|---|---|---|
| `check-sensitive.sh` | 可用 | 脱敏扫描：检查 `site/` 下是否混入敏感词 |
| `html2mdx.py` | v0.1 实现 | 解析原型 HTML 的 `.card` 结构，产出 MDX 骨架（只做机械转换，语义改写人工完成） |
| `check-links.sh` | v1.0 实现 | 全站外链与站内锚点检查 |
| `sensitive-words.txt` | 本地，不入库 | 敏感词表，从 `.example` 复制后按需扩充 |

## 用法

```bash
# 首次：准备词表
cp scripts/sensitive-words.txt.example scripts/sensitive-words.txt

# 迁移后扫描
./scripts/check-sensitive.sh
```

`check-sensitive.sh` 在 CI 中可选启用（词表不进库，故 CI 用内置的最小词表）。
