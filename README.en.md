# AIDC Observability · 智算网络可观测指南

> From lossy IDC networks to lossless AI fabrics: mechanisms, telemetry, and runnable labs.

**Status: v0.4** — 23 lessons + appendices A/B published, 12 labs shipped (10 L1 + 2 L2 real-topology).
Site: <https://lerixhe.github.io/aidc-observability/>

> Readable (v0.1), runnable (v0.2), complete (v0.3), advanced (v0.4).
> The two L2 labs run in two modes: the default mode needs no Docker, only Go.
> `make deploy` is the step that requires containerlab.

> The course text is in Chinese — [中文 README](README.md). This English README exists so
> that non-Chinese readers can evaluate the project quickly; the labs, their code and
> their outputs are language-neutral.

---

## What problem this solves

AI training fabrics (AIDC — AI data centers) overturn the assumptions behind
traditional network observability:

| | Traditional IDC | AI fabric |
|---|---|---|
| Traffic model | North-south mice flows, random arrival | East-west elephant flows, periodic AllReduce bursts |
| Loss semantics | Background noise, silently retransmitted by TCP | The incident itself, expensive to recover from |
| Cost of a fault | One server, one batch of requests | Hours × thousands of GPUs, 30-minute rollback |
| Telemetry granularity | Minute-level SNMP polling | Second / sub-second streaming subscriptions |

Copying an IDC dashboard over will fail. This project takes the new rules apart
layer by layer: the RDMA / RoCEv2 causal chain, PFC / ECN mechanisms and their
failure modes, gNMI / INT telemetry collection, metric definitions, and RCA automation.

## What makes it different

Plenty of material explains RoCE and PFC. Very little approaches the problem from
the **observability** side, and almost nothing ships **hands-on labs that reproduce
a PFC storm or incast in containers**.

The course explains mechanisms; `labs/` gives you code that reproduces them — that gap
is what this project tries to fill.

## Repository layout

```
.
├── docs/                  Planning and decisions (start with PROJECT.md)
│   ├── PROJECT.md         Context: positioning, audience, stack, milestones, risks
│   ├── OUTLINE.md         23-lesson outline + mapping to the original prototype
│   ├── MIGRATION.md       Migration and de-identification rules
│   └── DECISIONS.md       ADR log
├── site/                  VitePress site (live since v0.1)
├── labs/                  Go / container labs (12: 10 L1 + 2 L2)
├── reference/             Original prototype (local only, not committed)
└── scripts/               Migration, link check, style check
```

## Roadmap

| Version | Content | Status |
|---|---|---|
| v0.0 | Planning docs, directory skeleton, prototype archived | Done |
| v0.1 | 16 lessons as Markdown, docs site, GitHub Pages live | Done |
| v0.2 | First 6 L1 labs (no privileged deps), devcontainer, labs CI | Done |
| v0.3 | 8 new lessons (intro, environment, superpod, INT, pipeline, SLI/SLO, MTTR, end-to-end case) + 3 L1 labs + appendices A/B | Done |
| v0.4 | Advanced labs: containerlab topology (05-clos-topo), fault injection (03-fault-inject), end-to-end runbook (11-case-study) | **Current** |
| v1.0 | Polish: link checker, bilingual README, announcement | In progress |

See [docs/PROJECT.md](docs/PROJECT.md) for details (Chinese).

## Labs at a glance

```bash
cd labs/00-setup && make run          # which labs can this machine run?
cd labs/02-pfc-sim && make compare    # PFC / ECN / DCQCN queue model
cd labs/05-clos-topo && make run      # why ECMP polarizes at low flow counts
cd labs/11-case-study && make run     # two runbook paths, 40 vs 79 minutes
```

Every lab reads and writes plain text; `expected/` holds byte-stable samples that CI
diffs against, so any behavior change shows up as a diff instead of slipping through.

| Lab | Layer | Lesson | What it shows |
|---|---|---|---|
| `00-setup` | L1 | 01 | Environment probe and lab inventory |
| `01-traffic-model` | L1 | 02 | Mice flows vs synchronized AllReduce bursts |
| `02-pfc-sim` | L1 | 05, 06 | Watermarks, pause propagation, DCQCN closed loop |
| `03-fault-inject` | L2 | 07 | Four failure modes: injection plans, telemetry fingerprints, classifier |
| `04-nccl-trace` | L1 | 08 | Rebuild the traffic matrix from NCCL logs, find stragglers |
| `05-clos-topo` | L2 | 09 | Flow count and hash policy decide ECMP polarization |
| `06-gnmi-mock` | L1 | 11 | SAMPLE vs ON_CHANGE fidelity |
| `07-int-path` | L1 | 12 | INT hop-by-hop metadata, sampling capture rate |
| `08-pipeline` | L1 | 14 | Telemetry pipeline: backpressure and query fidelity |
| `09-slo` | L1 | 15 | Three availability definitions, error budgets, burn rate |
| `10-rca` | L1 | 16 | Alert correlation and graph-based root cause scoring |
| `11-case-study` | L1 | 21 | Ten-step runbook and an MTTR time budget |

The L2 labs declare their boundary up front: containers cannot emit real PFC pause
frames, so those columns come from a model. Each README says which is which and how
to verify on real hardware.

## About reference/

`reference/archive-aidc/` is an early private prototype (17 self-contained HTML lessons).
The structure and pedagogy of this project grew out of it.

That directory **contains personal narrative and former-employer information. It is
excluded via `.gitignore` and never committed.** Public content is rewritten according
to [docs/MIGRATION.md](docs/MIGRATION.md) before it reaches `site/`.

## License

- Text and course content: [CC BY-SA 4.0](LICENSE)
- `labs/` and `scripts/` code: [MIT](LICENSE-CODE)

## Contributing

The project is at v1.0-dev; content is still under review. Corrections are welcome —
please cite the lesson and a verifiable source:

- Mechanism claims need a checkable source (RFC / IEEE standard / paper / vendor docs)
- De-identification red lines are in section 3 of [docs/MIGRATION.md](docs/MIGRATION.md);
  changes involving internal information will not be merged
- Local development: `cd site && pnpm install && pnpm dev`
