# Logr Level Cache

This document describes the level caching system in logr: what is implemented, the
measurements behind it, and the analysis of production usage patterns that drove it.

## Current Implementation

**Location:** `levelcache.go`

A single implementation, `atomicLevelCache`, is used for both caching layers:

- **Top-level cache** (`Logr.lvlCache`, created in `logr.go`) answers "does ANY target
  want this level?" for `Logr.IsLevelEnabled`.
- **Per-target cache** (`TargetHost.lvlCache`, created in `target.go`) answers "does THIS
  target want this level?" and is consulted during fanout.

Each cache is a `[MaxLevelID + 1]atomic.Uint32` (256KB) plus a generation counter. Each
entry packs a 30-bit generation with two status bits:

```
31                              2 1 0
+--------------------------------+-+-+
|          generation            |S|E|
+--------------------------------+-+-+
```

An entry whose generation does not match the cache generation is stale, which makes
`clear()` an O(1) generation increment: no entry is touched and no allocation occurs.
The array is only physically zeroed during the generation rollover reset, reached after
roughly a billion clears.

Reads are two atomic loads with no lock and no allocation, so they scale with
core count rather than contending on a shared lock or map.

`UseArrayLevelCache` and `UseSyncMapLevelCache` remain in `options.go` as deprecated
no-ops so existing callers keep compiling.

## Why This Design

**Access patterns:**

- Read frequency: millions of times per second (every `Log()` call).
- Write frequency: only on config changes and target add/remove.
- Read/write ratio: roughly 1,000,000,000 : 1.
- Concurrency: high, many goroutines logging simultaneously.

**Typical log patterns:**

- 3-5 tags per log item (for example "LDAPError", "error", "auth", "connection").
- 3-5 log targets configured per application.
- Each tag creates a separate log record via `LogM()`, and each record is checked against
  every target during fanout, so a single log call performs 9-25 filter checks.

**Real-world level distribution from Mattermost:**

- 32 levels in use, with IDs 0-7, 10-11, 100-103, 130-132, 140-144, 200-204, 300-304.
- Sparse, with large gaps (204 to 65,535 unused).

A cache read therefore has to be fast under concurrency above all else, and invalidation
has to be cheap because it happens on every `AddTarget` and every `ResetLevelCache`.

## History

Two earlier implementations were replaced:

1. **arrayLevelCache** - a `[MaxLevelID + 1]LevelStatus` array guarded by a `sync.RWMutex`.
   The original default. Reads contend on the mutex's reader counter, so they get slower
   as concurrency rises.
2. **syncMapLevelCache** - sparse storage in a `sync.Map`, the default after the first
   round of optimizations. Reads are lock-free but slower than an array index, writes
   allocate, and `clear()` stored an empty entry for every one of the 65,535 ids.

Both were removed in September 2026 in favour of `atomicLevelCache`, which is faster on
every measured axis. The comparison below is the measurement that decided it.

## Measured Performance

Medians of 6 runs, `go1.26.3 linux/amd64`, 13th Gen Intel Core i9-13900K (32 threads).
All three implementations were measured through the `levelCache` interface, so the
numbers are directly comparable.

| Benchmark | GOMAXPROCS | syncMap | array | atomic |
|---|---|---|---|---|
| Get, cache hit | 1 | 9.25 ns | 7.93 ns | **1.43 ns** |
| Get, cache miss | 1 | 9.30 ns | 7.89 ns | **1.14 ns** |
| Get, spread over 256 ids | 1 | 16.71 ns | 7.89 ns | **1.45 ns** |
| Put | 1 | 76.99 ns (51 B, 2 allocs) | 17.57 ns | **3.86 ns** |
| Clear | 1 | 5.35 ms (3.5 MB, 131k allocs) | 15.23 µs | **6.66 ns** |
| `IsLevelEnabled`, end to end | 1 | 10.25 ns | 9.18 ns | **2.39 ns** |
| Get, cache hit | 32 | 1.05 ns | 22.13 ns | **0.08 ns** |
| Get, 99:1 read/write mix | 32 | 5.99 ns | 70.19 ns | **1.26 ns** |
| Get during concurrent clears | 32 | 0.82 ns | 25.47 ns | **0.08 ns** |
| `IsLevelEnabled`, end to end | 32 | 0.84 ns | 26.84 ns | **0.17 ns** |

Two results drove the decision beyond raw read speed:

- **The array cache does not scale.** Its reads are slower at 32 threads (22 ns) than
  single threaded (7.9 ns), because every reader touches the same `RWMutex` counter.
- **`clear()` was expensive.** Each syncMap clear cost 5.35 ms and 3.5 MB of garbage, and
  `resetLevelCacheLocked` clears the top-level cache plus every target's cache, so a
  single `AddTarget` with five targets cost roughly 30 ms.

Benchmarks for the surviving implementation live in `levelcache_bench_test.go`:

```
go test -vet=off -run '^$' -bench 'LevelCache|IsLevelEnabledCached' -benchmem -cpu 1,32 -count 6 .
```

## Known Gaps

- The `levelCache` interface now has a single implementer, and the indirect call is
  measurable: making `Logr.lvlCache` and `TargetHost.lvlCache` concrete takes
  `IsLevelEnabled` from 2.25 ns to 1.84 ns serial and 0.17 ns to 0.12 ns at 32 threads.
  The interface is kept as the seam for the per-tag cache described in
  `TAGGING-DESIGN.md`.
- `TargetHost.IsLevelEnabled` caches only `Enabled`, and on a cache hit it returns the
  caller's `Level` rather than the filter-resolved one (`target.go:141`). A caller that
  reaches the per-target cache while the top-level cache is cold can therefore record
  `Stacktrace: false` for a level whose filter asks for a stack trace. Fanout
  (`logr.go:464`) discards the returned level, so only `Logr.IsLevelEnabled` is affected.

## Notes

- This document covers integer-based level IDs. A string-based tag system is designed in
  `TAGGING-DESIGN.md`.
- Per-target caching becomes more valuable with string-based levels, not less.
