# Architecture

`blackhole-threats` is organised as a small set of focused components:

- `cmd/blackhole-threats`: process entrypoint, flag parsing, logging setup, signal wiring
- `internal/buildinfo`: build metadata injected by CI and release builds
- `internal/config`: YAML loading, feed definitions, BGP community parsing
- `internal/feed`: feed retrieval, plain-text and JSONL parsing, prefix summarisation
- `internal/bgp`: GoBGP lifecycle, route diffing, announce/withdraw orchestration
- `examples/`: operator-facing reference configuration
- `packaging/container/`: container rootfs and S6 service definitions
- `debian/`: packaging metadata for Debian-style builds

## Runtime Flow

1. Load YAML configuration from `blackhole-threats.yaml`.
2. Merge configured feeds with any extra `-feed` arguments.
3. Start the embedded GoBGP server.
4. Fetch and parse all configured feeds concurrently.
5. If a feed refresh fails for a community, keep the last good routes for that
   community instead of withdrawing on partial input.
6. Summarise prefixes and group them by community.
7. Diff the new route set against the current route set.
8. Announce new routes and withdraw stale routes.
9. Repeat on the configured refresh interval or immediately on `SIGUSR1`.

## Design Goals

- Keep the operator-facing workflow simple.
- Prefer conservative route handling when upstream feeds fail.
- Preserve familiar flag and config patterns where practical.
- Prefer small internal packages with clear ownership.
- Keep release automation first-party and easy to audit.
