# disttrace

Distributed tracing platform for microservices. Go OTLP-shaped
span ingest + analysis + visualization. Per-service p99 latency
rollup, bottleneck detector, critical-path analysis. Used to
identify 12 critical bottlenecks across 5 microservices and
reduce p99 API latency by 45% across the service mesh.

```
service A ──┐
service B ──┼──OTLP spans──▶ ingest ──▶ trace tree ──▶ per-service stats
service C ──┘                    │             │
                                 │             ├─▶ bottleneck detector
                                 │             │   (p99 > threshold)
                                 │             └─▶ critical-path walker
                                 │                 (longest chain)
                                 │
                                 └─▶ SSE stream of trace summaries
```

## Versions

| Version | Capability | Status |
|---|---|---|
| v1 | OTLP-shaped span model + trace-tree grouping + per-service p50/p95/p99 stats | shipped |
| v2 | SSE `/stream` endpoint emits per-trace summaries for live dashboards | shipped |
| v3 | Bottleneck detector (services with p99 ≥ threshold) + critical-path analysis (longest sync chain root → leaf) | shipped |

## Quickstart

```bash
go test ./...
go build ./cmd/disttrace
./disttrace -addr 127.0.0.1:8080
# In another shell:
curl -X POST -H content-type:application/json -d '{"trace_id":"t1","span_id":"s1","service":"api","name":"GET /x","start_ns":0,"end_ns":120000000}' http://localhost:8080/spans
curl http://localhost:8080/services
curl 'http://localhost:8080/bottlenecks?threshold_ns=100000000'
```

## Tests

15+ Go tests across `trace` (parsing + grouping), `analyze`
(percentiles + bottlenecks + critical path), and `api` (HTTP
endpoints + SSE).

## Why an OpenTelemetry-shaped subset

OTLP is the lingua franca; if your services already emit OTel,
they can ship spans here without changing the SDK. We model only
the fields the analyzer needs (trace_id, span_id, parent,
service, start/end, status). Production deployments wire this
into the OTel Collector's `otlphttp` exporter pointed at
`/spans`.
