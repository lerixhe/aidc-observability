# 03-fault-inject · 四大故障的注入剧本与遥测指纹
GO ?= go
S ?= storm

.PHONY: run plan sim classify test expected deploy inject inject-pfc xdp-build xdp-attach xdp-close xdp-open xdp-stats xdp-off observe destroy clean

IF ?= eth1
NODE ?= leaf2
PRIO ?= 3
COUNT ?= 10

# 默认：四个场景的计划 + 样本摘要 + 判读结论，不碰 Docker
run:
	$(GO) run .

# 只打印注入计划：这就是照着敲的剧本
plan:
	$(GO) run . -plan

# 生成样本 CSV 到 samples/，再自己判读一遍
sim:
	$(GO) run . -sim

classify:
	$(GO) run . -classify -dir samples

test:
	$(GO) test ./...

# 重新生成 expected/ 与 samples/（改了场景或判据才需要跑）
expected:
	@mkdir -p expected
	$(GO) run . -plan -scenario storm > expected/plan-storm.txt
	$(GO) run . -plan -scenario deadlock > expected/plan-deadlock.txt
	$(GO) run . -plan -scenario incast > expected/plan-incast.txt
	$(GO) run . -plan -scenario hol > expected/plan-hol.txt
	$(GO) run . -sim -dir samples > expected/sim.txt
	$(GO) run . -classify -dir samples > expected/classify.txt
	$(GO) run . > expected/report.txt

# ---- 以下三个目标需要 Docker + containerlab，L1 读者可以跳过 ----
deploy:
	containerlab deploy -t topo.clab.yml --reconfigure

# 注入真实 0x8808 PFC Pause 帧（自动选容器内 python3 或宿主 nsenter 两条路径）
inject-pfc:
	./scripts/inject_pfc.sh $(NODE) $(IF) $(PRIO) $(COUNT)

# 编译并挂载 XDP 反压闸门（需 clang + bpftool，Codespaces 已预装）
xdp-build:
	./scripts/xdp_gate.sh build

xdp-attach:
	./scripts/xdp_gate.sh attach $(NODE) $(IF)

xdp-close:
	./scripts/xdp_gate.sh close

xdp-open:
	./scripts/xdp_gate.sh open

xdp-stats:
	./scripts/xdp_gate.sh stats

xdp-off:
	./scripts/xdp_gate.sh off $(NODE) $(IF)

inject:
	$(GO) run . -apply -scenario $(S)

observe:
	@echo "按 README 第 7 节采计数器，落成 samples/live-$(S).csv，再跑 make classify"

destroy:
	-$(GO) run . -destroy
	-containerlab destroy -t topo.clab.yml --cleanup

clean:
	rm -rf out
