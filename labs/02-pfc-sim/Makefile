# 02-pfc-sim · 参数化队列模型
GO ?= go

.PHONY: run compare test expected clean

run:
	$(GO) run .

compare:
	$(GO) run . -compare -duration 1000

test:
	$(GO) test ./...

# 重新生成 expected/ 下的对照样本（改了模型才需要跑）
expected:
	@mkdir -p expected
	$(GO) run . -compare -duration 1000 > expected/compare.txt
	$(GO) run . -mode incast -duration 1000 | sed '/时序已写入/d' > expected/incast-pfc.txt

clean:
	rm -rf out
