# 11-case-study · 端到端排障剧本
GO ?= go

.PHONY: run steps live test expected clean

# 默认：最优路径与蛮干路径各走一遍，再给 MTTR 对照
run:
	$(GO) run .

# 完整剧本：十步，每步的判据都写出来
steps:
	$(GO) run . -steps

# 十步里哪几步能在容器拓扑里复刻
live:
	$(GO) run . -live

test:
	$(GO) test ./...

expected:
	@mkdir -p expected
	$(GO) run . > expected/compare.txt
	$(GO) run . -steps > expected/steps.txt
	$(GO) run . -live > expected/live.txt
	$(GO) run . -run best > expected/best.txt
	$(GO) run . -run naive > expected/naive.txt

clean:
	rm -rf out
