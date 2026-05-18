# Relayd

A lightweight reverse proxy in Go that stays minimal while showcasing how core reverse-proxy internals work using the standard library.

Relayd focuses on clean request forwarding, upstream reuse, simple load balancing, and graceful shutdown without trying to be a full API gateway or service mesh.

## Core Architecture

```/dev/null/arch.txt#L1-11
Client
  ↓
TCP Listener
  ↓
http.Server
  ↓
Router
  ↓
Backend Group
  ↓
Load Balancer (round-robin / random)
  ↓
ReverseProxy
  ↓
Shared Transport
  ↓
Upstream Backend
```

## Request Lifecycle

```/dev/null/lifecycle.txt#L1-9
Client request
  ↓
HTTP server accepts connection
  ↓
Router selects backend group
  ↓
Load balancer picks backend
  ↓
Reverse proxy forwards request
  ↓
Response is streamed back
```

## Features
- HTTP/1.1 reverse proxy
- Multiple upstreams with round-robin load balancing
- Shared `http.Transport` for connection pooling and keep-alive
- Streaming request/response forwarding
- Active health checks
- Graceful shutdown

## Requirements
- Go 1.20+ (see `go.mod`)

## Run
```/dev/null/run.sh#L1-1
go run ./cmd/relayd
```

## Build
```/dev/null/build.sh#L1-1
go build ./cmd/relayd
```

## Minimal benchmark
```/dev/null/bench.sh#L1-1
go test -run '^$' -bench='Benchmark(GetBackend|ServeHTTP)$' -benchmem ./internal/router
go test -run '^$' -bench=BenchmarkRoundRobin -benchmem ./internal/upstream
go test -run '^$' -bench=BenchmarkTransportSharing -benchmem -benchtime=10s -count=3 ./internal/proxy
```

These benchmarks measure route lookup (`GetBackend`), router+proxy request dispatch (`ServeHTTP`), and load balancer selection (`RoundRobin`).
Use `ns/op`, `B/op`, and `allocs/op` from the output as the observations in your post.

### Transport sharing result (interpretable summary)

From `BenchmarkTransportSharing` (`-benchtime=10s -count=3`):

| Mode | Avg time/op | Avg upstream_conns/op |
| --- | ---: | ---: |
| Shared transport | 19.15 us/op | 0.0000174 |
| New transport per proxy | 22.96 us/op | 0.0001205 |

Result: sharing one `http.Transport` across reverse proxies is about **20% faster** and creates about **6.9x fewer new upstream connections** under load.
