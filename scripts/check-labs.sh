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

# ---- 2. expected/ 与 samples/ 自校验 ----
note ""
note "== expected/ 自校验 =="
# 00-setup 的输出含主机信息（工具版本、平台），不适合跨机器比对，跳过
for lab in 01-traffic-model 02-pfc-sim 03-fault-inject 04-nccl-trace 05-clos-topo \
           06-gnmi-mock 07-int-path 08-pipeline 09-slo 10-rca 11-case-study; do
  dir="labs/$lab"
  [ -d "$dir/expected" ] || continue
  printf '  %-18s ' "$lab"
  if ! out="$( cd "$dir" && make -s expected 2>&1 )"; then
    note "❌ 生成失败"
    printf '%s\n' "$out" | tail -5 | sed 's/^/      /'
    fail=1
    continue
  fi

  # 03-fault-inject 的模型样本落在 samples/，与 expected/ 一样要锁住
  watched="$dir/expected"
  [ -d "$dir/samples" ] && watched="$watched $dir/samples"

  if git diff --quiet -- $watched; then
    note "✅"
  else
    note "❌ 与 expected/ 或 samples/ 不一致"
    git diff -- $watched | sed 's/^/      /'
    note "      确认是有意改动后：git add $watched"
    fail=1
  fi
done

# ---- 2b. L2 lab 的拓扑文件结构检查 ----
# 真拓扑要 Docker 才能 deploy，但 YAML 与计划的一致性用测试就能守住，
# 这部分在第一步的 go test 里已经跑过（topo_test.go）。
note ""
note "== L2 拓扑文件 =="
for f in labs/03-fault-inject/topo.clab.yml labs/05-clos-topo/topo.clab.yml; do
  [ -f "$f" ] || continue
  printf '  %-40s ' "$f"
  if grep -q $'\t' "$f"; then
    note "❌ 含 tab，containerlab 会读错缩进"
    fail=1
  elif ! grep -q '^name:' "$f"; then
    note "❌ 缺 name 字段"
    fail=1
  else
    note "✅"
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
