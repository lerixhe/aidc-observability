# 01-traffic-model · 到达形态对照
GO ?= go

.PHONY: run test expected clean

run:
	$(GO) run .

test:
	$(GO) test ./...

# 重新生成 expected/summary.txt（改了模型才需要跑）
expected:
	@mkdir -p expected
	$(GO) run . | sed '/时序已写入/d' > expected/summary.txt

clean:
	rm -rf out
