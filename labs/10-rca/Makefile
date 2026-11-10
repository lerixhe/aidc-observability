# 10-rca · 告警收敛与图打分根因定位
GO ?= go

.PHONY: run topo test expected clean

run:
	$(GO) run .

topo:
	$(GO) run . -topo

test:
	$(GO) test ./...

# 重新生成 expected/summary.txt（改了拓扑、算法或注入才需要跑）
expected:
	@mkdir -p expected
	$(GO) run . -trials 200 -seed 42 > expected/summary.txt

clean:
	rm -rf out
