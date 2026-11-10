#!/usr/bin/env bash
# 脱敏扫描：确保 site/ 下未混入前雇主信息、面试语境与内部系统名。
# 用法：./scripts/check-sensitive.sh [目录]
set -uo pipefail

TARGET="${1:-site}"
WORDS_FILE="$(dirname "$0")/sensitive-words.txt"

if [[ ! -d "$TARGET" ]]; then
  echo "跳过：目录 $TARGET 不存在（v0.1 前为空是正常的）"
  exit 0
fi

if [[ -f "$WORDS_FILE" ]]; then
  PATTERN="$(grep -v '^\s*#' "$WORDS_FILE" | grep -v '^\s*$' | paste -sd'|' -)"
  echo "使用本地词表：$WORDS_FILE"
else
  # CI 无词表时的兜底最小集：只含可用于公开的通用词
  # （前雇主名 / 内部系统名属于红线词，只写本地 sensitive-words.txt 或 CI secret）
  PATTERN='面试官|一面|二面|三面|薪资|JD 要求'
  echo "未找到本地词表，使用内置最小词表"
fi

# 排除依赖与构建产物，避免命中第三方英文文本
HITS="$(grep -rniE "$PATTERN" "$TARGET" \
  --exclude-dir=node_modules --exclude-dir=dist --exclude-dir=.astro \
  --exclude-dir=.git 2>/dev/null || true)"

if [[ -n "$HITS" ]]; then
  echo "❌ 发现敏感词，请修正后再提交："
  echo "$HITS"
  exit 1
fi

echo "✅ 脱敏扫描通过：$TARGET"
