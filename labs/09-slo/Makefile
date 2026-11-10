# 09-slo · 三种 SLO 口径与错误预算
GO ?= go

.PHONY: run test expected clean

run:
	$(GO) run .

test:
	$(GO) test ./...

# 重新生成 expected/summary.txt（改了流量、事故或口径才需要跑）
expected:
	@mkdir -p expected
	$(GO) run . > expected/summary.txt

clean:
	rm -rf out
