#!/usr/bin/env bash
# labs 统一检查：每个 lab 跑 go vet + go test，然后核对 expected/ 里的对照样本。
#
# 本地用法：scripts/check-labs.sh
# CI 用法：见 .github/workflows/labs.yml
#
# 约定：改模型就重新生成 expected（cd labs/<lab> && make expected），
# 让改动在 diff 里显式出现，而不是悄悄放过。

set -uo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root" || exit 1

fail=0
note() { printf '%s\n' "$*"; }
bad() { printf '❌ %s\n' "$*"; fail=1; }

# ---- 1. 每个 lab 过 vet + test ----
note "== go vet + go test =="
for mod in labs/*/go.mod; do
  dir="$(dirname "$mod")"
  name="$(basename "$dir")"
  printf '  %-18s ' "$name"
  if output="$(cd "$dir" && go vet ./... 2>&1 && go test ./... 2>&1)"; then
    note "✅"
  else
    note "❌"
    printf '%s\n' "$output" | sed 's/^/      /'
    fail=1
  fi
done

# ---- 2. expected/ 自校验 ----
note ""
note "== expected/ 自校验 =="
# 00-setup 的输出含主机信息（工具版本、平台），不适合跨机器比对，跳过
for lab in 01-traffic-model 02-pfc-sim 04-nccl-trace 06-gnmi-mock 10-rca; do
  dir="labs/$lab"
  [ -d "$dir/expected" ] || continue
  printf '  %-18s ' "$lab"
  if ! out="$( cd "$dir" && make -s expected 2>&1 )"; then
    note "❌ 生成失败"
    printf '%s\n' "$out" | tail -5 | sed 's/^/      /'
    fail=1
    continue
  fi
  if git diff --quiet -- "$dir/expected"; then
    note "✅"
  else
    note "❌ 与 expected/ 不一致"
    git diff -- "$dir/expected" | sed 's/^/      /'
    note "      确认是有意改动后：git add $dir/expected"
    fail=1
  fi
done

# ---- 3. 00-setup 体检（只跑不比对） ----
note ""
note "== 00-setup 体检 =="
if ( cd labs/00-setup && go run . >/dev/null 2>&1 ); then
  note "  ✅ 环境探测正常"
else
  bad "00-setup 执行失败"
fi

note ""
if [ "$fail" -eq 0 ]; then
  note "✅ labs 全部通过"
else
  note "❌ 有失败项，见上方输出"
fi
exit "$fail"
