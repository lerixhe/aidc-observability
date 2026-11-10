# 06-gnmi-mock · 订阅策略保真度
GO ?= go

.PHONY: run serve paths test expected clean

run:
	$(GO) run .

# 只起 mock 设备，用 curl 或浏览器观察
serve:
	$(GO) run . -serve

paths:
	$(GO) run . -paths

test:
	$(GO) test ./...

# 重新生成 expected/summary.txt（改了信号或路径树才需要跑）
expected:
	@mkdir -p expected
	$(GO) run . > expected/summary.txt

clean:
	rm -rf out
