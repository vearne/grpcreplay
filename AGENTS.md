# AGENTS.md — grpcreplay

gRPC traffic capture and replay tool. Single Go module, single CLI binary (`grpcr`).
Module: `github.com/vearne/grpcreplay`, Go 1.25, requires `CGO_ENABLED=1` (libpcap).

## Build & Test

```bash
# Prerequisites: install libpcap
#   macOS:  brew install libpcap
#   Ubuntu: sudo apt-get install -y libpcap-dev

make build                              # CGO_ENABLED=1 go build with ldflags
go test -v ./...                       # unit tests (no external services needed)
go test -v ./http2/...                 # single package tests
golangci-lint run -v --modules-download-mode=mod  # lint (v2 config)
```

CI removes `example/` and `test/` directories before running tests/lint. Those dirs contain standalone integration programs and example code, not `go test` suites.

## Lint Rules (.golangci.yml v2)

Active linters: `copyloopvar errcheck gocognit(30) gocyclo(20) govet ineffassign staticcheck unused`.
Excluded paths: `test/`, `example/`, `third_party/`, `builtin/`.
`govet` has `fieldalignment` disabled.

## Architecture

```
main.go           # CLI entry, flag parsing, wires components
config/           # AppSettings struct, custom flag types (MultiStringOption)
biz/              # Core: Emitter (input→filter→output pipeline), Plugin registry (reflect-based), Filter chain, RateLimit
biz/itf.go        # Plugin interfaces: PluginReader, PluginWriter, Limiter
plugin/           # Concrete I/O plugins (raw socket, file dir, RocketMQ, gRPC forward, stdout, tcp_kill)
protocol/         # Message/Protocol types + codecs (simple, json)
http2/            # HTTP/2 frame parsing, HPACK, state machine, PBFinder (reflection + local .proto)
filter/           # Include/exclude method-match filters
buffpool/         # Shared byte buffer pool
consts/           # Build-time version info (injected via -ldflags -X)
util/             # StringSet, goroutine-safe buffer, net helpers
```

Data flow: Input plugins → `Emitter.CopyMulty()` → Filter chain → Rate limiter → Output plugins.

Plugin registration uses `reflect.Value.Call` in `biz/plugins.go` — constructors are called via reflection with variable args.

## Key Constraints

- **Only h2c** (plain-text HTTP/2). No TLS support yet.
- **Only Unary RPC**. Streaming is not supported.
- Raw socket capture (`--input-raw`) requires root/sudo.
- Client and server must be on **different hosts** (uses tcpkill to force reconnection).
- Version, BuildTime, GitTag are injected at build time via `-ldflags -X github.com/vearne/grpcreplay/consts.*`.

## Code Conventions

- Comments and doc strings are written in **Chinese**.
- Logging: `github.com/vearne/simplelog` (alias `slog`). Level set via `SIMPLE_LOG_LEVEL` env var (`debug|info|warn|error`).
- Errors: `github.com/pkg/errors` for wrapping.
- Test data lives in `testdata/` subdirectories within each package.
- No code generation step — `.proto` files are for test/example servers and parsed at runtime.
