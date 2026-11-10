# 04-nccl-trace · 从日志还原流量矩阵
GO ?= go

.PHONY: run gen test expected clean

run:
	$(GO) run .

gen:
	$(GO) run . -gen samples/nccl-ring8.log

test:
	$(GO) test ./...

# 重新生成 expected/summary.txt（改了模型或样本才需要跑）
expected:
	@mkdir -p expected
	$(GO) run . > expected/summary.txt

clean:
	rm -rf out
