# AIDC Observability · 智算网络可观测入门

> From lossy IDC networks to lossless AI fabrics: mechanisms, telemetry, and runnable labs.

[![Open in GitHub Codespaces](https://github.com/codespaces/badge.svg)](https://codespaces.new/lerixhe/aidc-observability)

**26 lessons + appendices A/B, 12 labs shipped (10 L1 + 2 L2 real topology).** Site: <https://lerixhe.github.io/aidc-observability/> · [中文 README](README.md)

This is Book 2 of the series. Book 1 is [IDC Network Observability](https://github.com/lerixhe/idc-network-observability/):
collection, metrics, reading curves, locating faults on lossy networks, written for backend
developers entering the field. Read that one first.

The course text is in Chinese. This English README exists so non-Chinese readers can evaluate the
project quickly; the labs, their code and their outputs are language-neutral.

---

## What problem this solves

AI training fabrics (AIDC — AI data centers, also called AI fabric or high-performance networking)
overturn the assumptions behind traditional network observability:

| | Traditional IDC | AI fabric |
|---|---|---|
| Traffic model | North-south mice flows, random arrival | East-west elephant flows, periodic AllReduce bursts |
| Loss semantics | Background noise, silently retransmitted by TCP | The incident itself, expensive to recover from |
| Cost of a fault | One server, one batch of requests | Hours × thousands of GPUs, 30-minute rollback |
| Telemetry granularity | Minute-level SNMP polling | Second / sub-second streaming subscriptions |

Copying an IDC dashboard over will fail. The course takes the new rules apart layer by layer:
the RDMA / RoCEv2 causal chain, PFC / ECN mechanisms and their failure modes, gNMI / INT
collection semantics, metric definitions, and RCA automation.

## What makes it different

Plenty of material explains RoCE and PFC. Very little approaches the problem from the
**observability** side, and almost nothing ships **hands-on labs that reproduce a PFC storm
or incast in containers**. The course explains mechanisms; `labs/` gives you code that
reproduces them.

## Course outline

| Part | Lessons | Topic |
|---|---|---|
| Start | opening 00–01 | Metrics all green, job still slow; getting the labs running |
| 1 · Paradigm shift | 02–04 | Why IDC habits fail, RDMA kernel bypass, RoCEv2 and DCB, adaptive routing and reordering |
| 2 · Lossless mechanics | 05–07 | PFC hop-by-hop backpressure, ECN and DCQCN, PFC failure modes |
| 3 · Traffic and topology | 08–10 | NCCL collectives, Clos and rail, superpods and scale-up, NVLink link faults |
| 4 · Telemetry | 11–14 | gNMI subscriptions, INT, optical modules, the data pipeline |
| 5 · Analysis and operations | 15–19 | Metric definitions and SLOs, AIOps root-cause, time-series storage, LLM-assisted ops, MTTR |
| 6 · Selection | 20–22 | IB vs RoCE, UEC, an end-to-end incident case, self-assessment |
| Appendices | A / B | Glossary, references |

## Labs

12 labs under `labs/`, in two tiers:

| Tier | Requires | Count | Property |
|---|---|---|---|
| L1 data-plane model | Go only | 10 | `go run .`, no privileges, no Docker |
| L2 real topology | Docker + containerlab | 2 | Real topology and fault injection; the default mode still needs nothing |

`03-fault-inject` and `05-clos-topo` run in two modes: `make run` prints the injection plan and
model conclusions, `make deploy` brings up containerlab. Real pause frames work too:

```bash
cd labs/03-fault-inject
make deploy          # 6-node topology
make inject-pfc      # inject genuine 0x8808 PFC pause frames onto a veth
docker exec clab-fault-inject-h3 tcpdump -i eth1 ether proto 0x8808 -c 3 -xx
```

PFC and ECN are hardware behavior; a software model cannot be behaviorally equivalent. Every L1
lab README states that boundary up front and gives a path to verify on real gear. `expected/`
holds byte-stable samples that CI diffs against, so a behavior change shows up as a diff instead
of slipping through.

## Repository layout

```
.
├── site/                  VitePress site (content under site/docs, one directory per part)
├── labs/                  Go / container labs (12, one module each)
└── scripts/               Link, anchor and Mermaid checks; lab self-verification
```

## Run it

```bash
# Site
cd site && pnpm install && pnpm dev

# First lab (Go 1.22+)
cd labs/00-setup && make run

# Verify every lab
./scripts/check-labs.sh
```

The L2 topology labs need to create veth pairs and move network namespaces, which macOS cannot do.
Use Codespaces: the badge above leads to an environment with Go, Docker and containerlab already
installed. When Docker is unavailable the setup script skips containerlab and L1 labs still run.

## License

- Text and course content: [CC BY-SA 4.0](LICENSE)
- `labs/` and `scripts/` code: [MIT](LICENSE-CODE)

## Contributing

The content is published in full; corrections are welcome as issues or pull requests. Name the
lesson and attach a checkable source:

- Mechanism claims need a verifiable source (RFC / IEEE standard / paper / vendor docs)
- Examples use RFC 5737 documentation addresses. Do not submit real device configurations,
  internal addresses, or unpublished architecture details
- Local development: `cd site && pnpm install && pnpm dev`
