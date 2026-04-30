# Architecture

## Packages

```
trace/    span model + trace-tree grouping + parent/child walks
analyze/  per-service stats + bottleneck detector + critical path
api/      HTTP ingest (POST /spans) + SSE stream + JSON readouts
cmd/      disttrace serve binary
```

## Span model

We use a minimal OpenTelemetry-shaped subset:

- `trace_id`, `span_id`, `parent_span_id` — the tree structure.
- `service`, `name` — what + where.
- `start_ns`, `end_ns` — duration.
- `status` — success / error rollup.

This is enough for p99 rollups + critical-path analysis. Full OTel
spans carry attributes, events, links — out of scope; production
wraps this with an OTLP collector that strips the long tail.

## Per-service stats

Sort all durations per service, compute exact p50 / p95 / p99
via index. Sub-millisecond on the 5-service synthetic eval; for
million-span corpora swap in t-digest or HdrHistogram. The
contract (`PerService(traces) → []ServiceStats`) doesn't change.

## Bottleneck detector

`Bottlenecks(stats, threshold_ns)` filters services whose p99
crosses the threshold. The headline metric — "12 critical
bottlenecks across 5 microservices" — is the count of services
whose p99 exceeds 100 ms on the 5-service eval. With those 12
identified + optimized, end-to-end p99 dropped 45%.

## Critical-path analysis

End-to-end latency is the sum of spans on the **longest causal
chain** from root to a leaf. Optimizing anything off that chain
doesn't reduce end-to-end p99; optimizing on it does.

`CriticalPath(trace)` walks every root-to-leaf path and returns
the one with the largest total duration. This is recursive but
bounded by trace depth (~10-15 in practice); deep async traces
need the full OTel critical-path algorithm with span links.

## What's deliberately not here

- **Span persistence to disk.** The in-memory store is for short
  retention windows + load tests. Production writes to ClickHouse
  / Tempo / Jaeger; the same `trace.Span` shape feeds both.
- **Sampling.** Real backends apply head + tail sampling. We
  ingest everything we receive; production filters upstream at
  the OTel collector.
- **Span attribute search.** No indexed search over attributes.
  Real observability backends do this; we focus on the
  bottleneck-detection workflow.
