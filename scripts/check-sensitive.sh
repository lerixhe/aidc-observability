#!/bin/sh
# 脱敏扫描：确保 site/ 下未混入前雇主信息、面试语境与内部系统名。
#
# 用法：
#   ./scripts/check-sensitive.sh                # 扫 site/
#   ./scripts/check-sensitive.sh site labs      # 扫多个目录
#   sh scripts/check-sensitive.sh               # 用 dash / busybox sh 跑也成立
#
# 这份脚本刻意写成 POSIX sh：早先的版本用了 bash 的 [[ ]] 与 GNU grep 的 \s，
# 一旦有人用 `sh scripts/check-sensitive.sh`（容器镜像里很常见）就会报
# "[: not found" 之类的错，或者更糟 —— 静默地扫不出东西。现在：
#   * 只用 POSIX 语法与字符类（[[:space:]]），不依赖 bash 扩展
#   * 缺 paste 时退回 sed 拼装，缺 grep -I 时退回 --exclude-dir + 后缀过滤
#   * 词表全被注释掉时**不会**退化成空 pattern —— 空 pattern 会命中每一行
#
# 退出码：0 通过或跳过；1 命中敏感词；2 环境依赖缺失。
set -eu

SELF_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
WORDS_FILE="$SELF_DIR/sensitive-words.txt"

die() {
	echo "$*" >&2
	exit 2
}

command -v grep >/dev/null 2>&1 || die "check-sensitive.sh：需要 grep，但找不到"

# ---- 待扫目录：默认 site，可传多个；不存在的目录跳过（v0.1 前为空是正常的）----
if [ "$#" -eq 0 ]; then
	set -- site
fi

TARGETS=""
for d in "$@"; do
	if [ -d "$d" ]; then
		TARGETS="$TARGETS $d"
	else
		echo "跳过：目录 ${d} 不存在"
	fi
done

if [ -z "${TARGETS# }" ]; then
	echo "✅ 没有可扫的目录，视为通过"
	exit 0
fi

# ---- 组装词表 pattern ----
# 词表一行一个词；# 开头与空行忽略。用 POSIX 字符类，不用 \s。
PATTERN=""
PATTERN_SRC="内置最小词表"
if [ -f "$WORDS_FILE" ]; then
	# shellcheck disable=SC2086
	WORDS=$(grep -v -e '^[[:space:]]*#' -e '^[[:space:]]*$' "$WORDS_FILE" || true)
	if [ -n "$WORDS" ]; then
		if command -v paste >/dev/null 2>&1; then
			PATTERN=$(printf '%s\n' "$WORDS" | paste -sd'|' -)
		else
			PATTERN=$(printf '%s\n' "$WORDS" | sed -e 's/[\/&]/\\&/g' | tr '\n' '|' | sed -e 's/|$//')
		fi
		PATTERN_SRC="${WORDS_FILE}（$(printf '%s\n' "$WORDS" | wc -l | tr -d ' ') 条）"
		echo "使用本地词表：${PATTERN_SRC}"
	fi
fi

if [ -z "$PATTERN" ]; then
	# 兜底最小集：只含可用于公开的通用词。
	# 前雇主名 / 内部系统名属于红线词，只写本地 sensitive-words.txt 或 CI secret。
	PATTERN='面试官|一面|二面|三面|薪资|JD 要求'
	if [ -f "$WORDS_FILE" ]; then
		echo "词表 ${WORDS_FILE} 里没有有效词条，退回内置最小词表"
	else
		echo "未找到本地词表，使用内置最小词表"
	fi
fi

# ---- 扫描 ----
# grep 的 -E 是 POSIX；-I（跳过二进制）不是每个平台都有，探测后决定用不用。
GREP_BIN="grep"
if grep -I --version >/dev/null 2>&1 || printf 'x\n' | grep -I x >/dev/null 2>&1; then
	GREP_BIN="grep -I"
fi

# shellcheck disable=SC2086
HITS=$($GREP_BIN -rniE \
	--exclude-dir=node_modules --exclude-dir=dist --exclude-dir=cache \
	--exclude-dir=.git --exclude-dir=__pycache__ \
	-e "$PATTERN" $TARGETS 2>/dev/null || true)

if [ -n "$HITS" ]; then
	echo "❌ 发现敏感词，请修正后再提交："
	printf '%s\n' "$HITS"
	echo
	echo "命中 $(printf '%s\n' "$HITS" | wc -l | tr -d ' ') 行。词表来源：${PATTERN_SRC}"
	exit 1
fi

echo "✅ 脱敏扫描通过：${TARGETS}"
