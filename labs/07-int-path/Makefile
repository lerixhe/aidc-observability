# 07-int-path · INT 逐跳元数据的编码与判读
GO ?= go

.PHONY: run csv test expected clean

run:
	$(GO) run .

# 逐跳明细写到 out/hops.csv，便于画图
csv:
	$(GO) run . -csv

test:
	$(GO) test ./...

# 重新生成 expected/summary.txt（改了场景、判读阈值或采样参数才需要跑）
expected:
	@mkdir -p expected
	$(GO) run . > expected/summary.txt

clean:
	rm -rf out
