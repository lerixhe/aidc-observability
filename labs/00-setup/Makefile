# 00-setup · 环境自检
GO ?= go

.PHONY: run try test expected clean

run:
	$(GO) run .

# 体检 + 对每个可跑的 L1 lab 跑 vet/test
try:
	$(GO) run . -try

test:
	$(GO) test ./...

# 重新生成 expected/summary.txt（新增 lab 或改登记表后跑）
# 主机相关的行（仓库路径、平台、工具版本）做脱敏，跨机器才能对照
expected:
	@mkdir -p expected
	$(GO) run . \
		| sed -E 's#^仓库 .*#仓库 <repo>#' \
		| sed -E 's#^平台 .*#平台 <os/arch> · <n> 核#' \
		| sed -E 's#^(go|python3|docker|containerlab)( +)[0-9][0-9.]*#\1\2<ver>#' \
		> expected/summary.txt

clean:
	rm -rf out
