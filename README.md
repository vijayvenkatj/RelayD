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
