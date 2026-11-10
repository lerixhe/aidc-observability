# 08-pipeline · 秒级遥测落盘与查询保真度对比
GO ?= go

.PHONY: run csv test expected clean

run:
	$(GO) run .

# 各档数据写到 out/，可画图对照
csv:
	$(GO) run . -csv

test:
	$(GO) test ./...

# 重新生成 expected/summary.txt（改了信号、档位或缓冲参数才需要跑）
expected:
	@mkdir -p expected
	$(GO) run . > expected/summary.txt

clean:
	rm -rf out
