# 05-clos-topo · ECMP 极化
GO ?= go

.PHONY: run topo test expected deploy collect destroy clean

# 默认：三个实验（流数 / 口径 / 大象流），不需要 Docker
run:
	$(GO) run .

# 拓扑与地址表，照着敲路由
topo:
	$(GO) run . -topo

test:
	$(GO) test ./...

expected:
	@mkdir -p expected
	$(GO) run . > expected/experiments.txt
	$(GO) run . -topo > expected/topology.txt
	$(GO) run . -exp 1 -scan 1,2,4,8,16 > expected/scan-small.txt

# ---- 以下需要 Docker + containerlab ----
deploy:
	containerlab deploy -t topo.clab.yml --reconfigure

# 起一拨流量，再采 10 秒上行计数，看真实分布
collect:
	./scripts/collect.sh 10

destroy:
	-containerlab destroy -t topo.clab.yml --cleanup

clean:
	rm -rf out
