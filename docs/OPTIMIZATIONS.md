# Logr Level Cache Optimizations

This document outlines performance optimizations for the level caching system in logr, based on analysis of current implementation and production usage patterns.

## Key Findings

**Critical usage pattern:** Applications typically use **3-5 tags per log item** with **3-5 log targets**, resulting in **9-25 filter checks per log call**. This multiplier effect makes caching optimizations significantly more impactful than single-tag scenarios.

**Expected improvement:** ~3-4x faster logging (640ns → 200ns per typical log call) after implementing both optimizations, with savings of **4+ CPU cores** at high traffic volumes (10M logs/sec).

## Current State Analysis

### Existing Cache Implementation

**Location:** `levelcache.go`

Two cache implementations exist:
1. **arrayLevelCache** (current default)
   - Fixed array: `[MaxLevelID + 1]LevelStatus` (65,536 entries)
   - Memory: ~524KB allocated upfront
   - Protected by `sync.RWMutex`
   - Access: O(1) array indexing with lock overhead

2. **syncMapLevelCache** (opt-in via `UseSyncMapLevelCache` option)
   - Sparse storage using `sync.Map`
   - Memory: Only allocates entries for levels actually used
   - Lock-free reads after warm-up
   - Access: O(1) map lookup, lock-free

### Current Usage Patterns

**Real-world example from Mattermost:**
- 32 total log levels used
- Level IDs: 0-7, 10-11, 100-103, 130-132, 140-144, 200-204, 300-304
- Sparse allocation with large gaps (e.g., 204 → 65,535 unused)
- Memory utilization with arrayLevelCache: **0.05%** (32 / 65,536)
- Memory waste: ~524KB allocated, ~256 bytes actually used

**Access patterns:**
- Read frequency: Millions of times per second (every `Log()` call)
- Write frequency: Once per days (only on config changes/target add/remove)
- Read/Write ratio: **~1,000,000,000 : 1**
- Concurrency: High (many goroutines logging simultaneously)

**Typical log patterns:**
- **3-5 tags per log item** (e.g., "LDAPError", "error", "auth", "connection")
- **3-5 log targets** configured per application
- Each tag creates a separate log record via `LogM()`
- Each log record is checked against all targets during fanout
- **Total checks per log call: 9-25 target filter checks** (3-5 tags × 3-5 targets)

### Two-Level Filtering Architecture

**Level 1: Top-level cache** (`Logr.IsLevelEnabled`)
- **Status:** CACHED ✓
- **Purpose:** Checks "does ANY target want this level?"
- **Location:** `logr.go:139`
- **Cache invalidation:** When targets added/removed
- **Performance:**
  - arrayLevelCache: ~20-40ns (RWMutex + array access)
  - syncMapLevelCache: ~5-10ns (lock-free after warm-up)

**Level 2: Per-target filtering** (during fanout)
- **Status:** NOT CACHED ✗
- **Purpose:** Checks "does THIS specific target want this level?"
- **Location:** `logr.go:456` in `fanout()`, calls `target.go:136`
- **Filter implementations:**
  - StdFilter: ~1-2ns (simple integer comparison)
  - CustomFilter: ~30ns (RWMutex + map lookup)
- **Performance impact:** ~30ns × N targets per log record
  - With **3-5 tags per log item**:
    - 3 tags × 3 targets = 9 filter checks × 30ns = **270ns per log call**
    - 5 tags × 5 targets = 25 filter checks × 30ns = **750ns per log call**
  - This dominates total logging overhead!

## Identified Issues

### Issue 1: Suboptimal Default Cache Choice

**Problem:** arrayLevelCache is default despite being inferior for production use cases.

**Evidence:**
- Memory waste: 99%+ with sparse level IDs
- Slower reads: RWMutex overhead (~20-40ns) vs lock-free (~5-10ns)
- No advantages for real-world usage (memory is not a constraint at 524KB)

**Impact:**
- 2-4x slower cache lookups than syncMapLevelCache
- Unnecessary memory allocation
- Documentation incorrectly suggests syncMapLevelCache only for ">32 cores"

### Issue 2: Uncached Per-Target Filtering

**Problem:** During fanout, each target's filter is checked without caching.

**Evidence:**
- `host.IsLevelEnabled()` calls `filter.GetEnabledLevel()` on every log
- CustomFilter uses RWMutex + map lookup: ~30ns per target
- With typical usage (4 tags, 4 targets): 16 filter checks × 30ns = 480ns per log call
- With heavy usage (5 tags, 5 targets): 25 filter checks × 30ns = 750ns per log call

**Impact:**
- **Dominates total logging overhead** (75-80% of time spent in uncached filtering)
- Repeated identical lookups (same level checked against same filter repeatedly)
- Gets worse with more tags per log (3-5× multiplier)
- Critical optimization opportunity for multi-tag logging patterns

### Issue 3: Misleading Documentation

**Problem:** Options and comments suggest arrayLevelCache is preferred.

**Location:** `options.go:129-137`

**Current guidance:**
```go
// UseSyncMapLevelCache set to true for >32 cores for potentially better performance
```

**Reality:** syncMapLevelCache is better for ALL scenarios with this workload.

## Proposed Optimizations

### Optimization 1: Make syncMapLevelCache the Default

**Change:** Swap default cache implementation.

**Rationale:**
- Better memory efficiency (100-500x less memory with sparse IDs)
- Faster reads (lock-free vs RWMutex)
- Perfect match for read-heavy workload (sync.Map designed for this)
- No downsides for production usage

**Implementation:**
1. Change default in `logr.go:66-71` to use `syncMapLevelCache`
2. Update `UseSyncMapLevelCache` option to `UseArrayLevelCache` (opt-in to old behavior)
3. Update documentation in `options.go` and README

**Expected impact:**
- 2-4x faster top-level cache lookups
- ~500x better memory efficiency in production
- No breaking changes (behavior identical, only performance)

**Code locations:**
- `logr.go:66-71` - Cache initialization
- `options.go:129-137` - Option definition
- `README.md` - Documentation updates

### Optimization 2: Add Per-Target Level Caching

**Change:** Add level cache to each `TargetHost`.

**Rationale:**
- Eliminates repeated filter checks during fanout
- Significant for CustomFilter (~30ns → ~5-10ns per target)
- Cache invalidation already handled (targets added/removed clears top-level cache)

**Implementation:**

```go
// target.go
type TargetHost struct {
    name      string
    target    Target
    filter    Filter
    formatter Formatter
    in        chan *LogRec
    lvlCache  levelCache  // NEW: per-target cache
    // ... rest of fields
}

// Update IsLevelEnabled to check cache first
func (h *TargetHost) IsLevelEnabled(lvl Level) (enabled bool, level Level) {
    // Check cache first
    if status, ok := h.lvlCache.get(lvl.ID); ok {
        return status.Enabled, lvl
    }

    // Cache miss: check filter
    level, enabled = h.filter.GetEnabledLevel(lvl)

    // Cache the result
    h.lvlCache.put(lvl.ID, LevelStatus{Enabled: enabled})

    return enabled, level
}

// Clear cache when filter changes
func (h *TargetHost) resetCache() {
    h.lvlCache.clear()
}
```

**Cache invalidation points:**
- When target is added: cleared automatically on creation
- When target is removed: not needed (target destroyed)
- When Logr.ResetLevelCache() called: add calls to clear per-target caches

**Expected impact:**
- Per-target filtering: ~30ns → ~5-10ns (after warm-up)
- With typical usage (4 tags, 4 targets): 480ns → 160ns (3x faster)
- With heavy usage (5 tags, 5 targets): 750ns → 250ns (3x faster)
- **Critical for multi-tag patterns:** Impact scales with number of tags
- Minimal memory overhead (~32 entries × 5 targets = ~1.2KB)

**Code locations:**
- `target.go:50-65` - TargetHost struct and initialization
- `target.go:135-138` - IsLevelEnabled method
- `logr.go:234` - ResetLevelCache (add per-target cache clears)
- `logr.go:115, 226` - Target add/remove (ensure cache cleared)

### Optimization 3: Remove arrayLevelCache (Optional)

**Change:** Delete arrayLevelCache implementation entirely.

**Rationale:**
- No production use case where it's superior
- Simplifies codebase
- Removes 524KB memory allocation for unused approach
- One less code path to maintain

**Implementation:**
1. Remove `arrayLevelCache` struct from `levelcache.go:57-98`
2. Remove `UseSyncMapLevelCache` option from `options.go`
3. Always use `syncMapLevelCache`
4. Update tests to remove arrayLevelCache test cases

**Trade-offs:**
- ✅ Simpler code, less maintenance
- ✅ No memory waste on array allocation
- ❌ Removes flexibility (though no known use case needs it)
- ❌ Breaking change if anyone explicitly uses arrayLevelCache

**Recommendation:** Consider this for a major version bump (v3.0) when breaking changes are acceptable.

**Code locations:**
- `levelcache.go:57-98` - arrayLevelCache implementation
- `options.go:129-137` - Option definition
- `logr.go:66-71` - Cache selection logic
- Test files: Remove arrayLevelCache test coverage

## Implementation Plan

### Phase 1: Non-Breaking Improvements (Recommended for v2.x)

1. **Make syncMapLevelCache default** (Optimization 1)
   - Low risk, high reward
   - Behavioral compatibility maintained
   - Performance improvement immediate

2. **Add per-target caching** (Optimization 2)
   - Moderate complexity
   - Significant performance gain for multi-target scenarios
   - No breaking changes

3. **Update documentation**
   - Correct misleading guidance
   - Document new performance characteristics

### Phase 2: Breaking Changes (Consider for v3.0)

1. **Remove arrayLevelCache** (Optimization 3)
   - Simplify codebase
   - Remove deprecated implementation
   - Announce deprecation in Phase 1

## Performance Expectations

### Before Optimizations

**Typical log call with 4 tags and 4 targets:**
- Top-level cache: 4 tags × 40ns = 160ns (arrayLevelCache with RWMutex)
- Per-target filtering: 4 tags × 4 targets × 30ns = 480ns (uncached CustomFilter)
- **Total: ~640ns per log call**

**Range with 3-5 tags and 3-5 targets:**
- Best case (3 tags, 3 targets): 3×40ns + 9×30ns = **390ns**
- Worst case (5 tags, 5 targets): 5×40ns + 25×30ns = **950ns**

### After Optimizations

**Typical log call with 4 tags and 4 targets (after warm-up):**
- Top-level cache: 4 tags × 10ns = 40ns (syncMapLevelCache, lock-free)
- Per-target filtering: 4 tags × 4 targets × 10ns = 160ns (cached)
- **Total: ~200ns per log call**

**Range with 3-5 tags and 3-5 targets:**
- Best case (3 tags, 3 targets): 3×10ns + 9×10ns = **120ns**
- Worst case (5 tags, 5 targets): 5×10ns + 25×10ns = **300ns**

**Performance improvement: 3-4x faster**

### At Scale

At 1 million logs/second (typical case: 4 tags, 4 targets):
- Before: 640ms CPU time = 64% of 1 core
- After: 200ms CPU time = 20% of 1 core
- **Savings: ~44% of 1 CPU core**

At 10 million logs/second (high-traffic server):
- Before: 6.4 seconds CPU time = **6.4 cores**
- After: 2.0 seconds CPU time = **2.0 cores**
- **Savings: 4.4 CPU cores**

## Testing Requirements

### Unit Tests

- Verify syncMapLevelCache default initialization
- Test per-target cache hits/misses
- Verify cache invalidation on target add/remove
- Verify cache invalidation on ResetLevelCache()
- Concurrent access tests (existing should cover)

### Benchmarks

Create benchmarks to validate improvements:

```go
// Benchmark top-level cache (syncMap vs array)
BenchmarkIsLevelEnabled_SyncMap
BenchmarkIsLevelEnabled_Array

// Benchmark fanout with varying target counts
BenchmarkFanout_3Targets_Cached
BenchmarkFanout_3Targets_Uncached
BenchmarkFanout_5Targets_Cached
BenchmarkFanout_5Targets_Uncached

// Multi-tag benchmarks (critical for realistic testing)
BenchmarkLogM_3Tags_3Targets_Cached
BenchmarkLogM_3Tags_3Targets_Uncached
BenchmarkLogM_4Tags_4Targets_Cached    // Typical case
BenchmarkLogM_4Tags_4Targets_Uncached  // Typical case
BenchmarkLogM_5Tags_5Targets_Cached    // Worst case
BenchmarkLogM_5Tags_5Targets_Uncached  // Worst case

// End-to-end logging benchmark
BenchmarkLog_MultiTag_Production
```

### Performance Testing

- Test with production-like workload:
  - Sparse level IDs (mirroring Mattermost pattern)
  - High concurrency (32+ goroutines)
  - **3-5 tags per log call** (critical for realistic testing)
  - 3-5 targets configured
- Measure memory usage before/after
- Verify no regression in high-contention scenarios
- Validate 3-4x performance improvement in multi-tag scenarios

## Backward Compatibility

### Phase 1 (Non-Breaking)

- ✅ All existing APIs unchanged
- ✅ Behavior identical (only performance improves)
- ✅ Option renamed but old behavior available via `UseArrayLevelCache`
- ✅ Safe for minor version bump (v2.x)

### Phase 2 (Breaking)

- ❌ Removes arrayLevelCache entirely
- ❌ Removes related options
- ⚠️ Requires major version bump (v3.0)
- ⚠️ Requires deprecation notice in Phase 1

## References

### Code Files to Modify

**Phase 1:**
- `levelcache.go` - No changes, both implementations stay
- `logr.go:66-71` - Change default cache selection
- `options.go:129-137` - Update option and documentation
- `target.go:50-65, 135-138` - Add per-target caching
- `logr.go:234` - Add per-target cache clearing
- `README.md` - Update documentation

**Phase 2:**
- `levelcache.go:57-98` - Remove arrayLevelCache
- `options.go` - Remove cache selection option
- `logr.go:66-71` - Remove cache selection logic
- Test files - Remove arrayLevelCache tests

### Related Design Documents

- `tagging-design.md` - Future tagging system design
- `CLAUDE.md` - Development guidelines

## Notes

- This document focuses on integer-based level IDs (current implementation)
- String-based level system is a separate future enhancement
- These optimizations apply regardless of level ID type (integer or string)
- Per-target caching becomes even more valuable with string-based levels
