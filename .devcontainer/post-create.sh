#!/usr/bin/env bash
# Codespaces / devcontainer 初始化：预装 containerlab 并做一次环境自检。
#
# containerlab 需要 Docker 与 root 权限建 veth、改网络命名空间，所以只在
# Docker-in-Docker 可用时装；macOS 本机无法运行，那里跳过即可。
set -uo pipefail

echo "== 1. 检查 Docker =="
if ! command -v docker >/dev/null 2>&1; then
	echo "  未找到 docker，跳过 containerlab 安装（L1 实验不受影响）"
	exit 0
fi
if ! docker info >/dev/null 2>&1; then
	echo "  docker 守护进程未就绪，跳过 containerlab 安装"
	exit 0
fi
echo "  ✅ docker 可用"

if command -v containerlab >/dev/null 2>&1; then
	echo "== 2. containerlab 已存在：$(containerlab version | head -n 1) =="
else
	echo "== 2. 安装 containerlab =="
	# 官方安装脚本，失败不影响 L1 实验
	if bash -c "$(curl -sL https://get.containerlab.dev)" >/dev/null 2>&1; then
		echo "  ✅ containerlab 安装完成：$(containerlab version | head -n 1)"
	else
		echo "  ⚠ containerlab 安装失败，L2 实验（03-fault-inject / 05-clos-topo）将不可用"
		echo "    手动重试：bash -c \"\$(curl -sL https://get.containerlab.dev)\""
	fi
fi

echo "== 3. 实验环境自检 =="
cd "$(dirname "$0")/../labs/00-setup" && go run . || true

echo
echo "下一步："
echo "  cd labs/00-setup && make run     # 体检"
echo "  cd labs/02-pfc-sim && make compare"
echo "  cd labs/03-fault-inject && make deploy && make inject-pfc"
