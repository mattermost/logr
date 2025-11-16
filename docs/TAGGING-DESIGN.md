# Tagging Design for Logr

## Overview

This document outlines the design for replacing the current hierarchical log level system with a flexible tag-based filtering system. Tags provide more granular control over log output while maintaining full backward compatibility.

## Prerequisites

This design assumes the optimizations described in `OPTIMIZATIONS.md` have been implemented first:
- ✅ syncMapLevelCache is the default
- ✅ Per-target caching exists and is proven (via `TargetHost.lvlCache`)
- ✅ Performance baseline established (~640ns per log call with 4 levels, 4 targets)

The tag system adapts this proven caching pattern by using **hybrid string-to-integer interning with COW per-tag array caching** internally while maintaining a string-based API. Instead of caching tag combinations (which requires key generation and sorting), we cache the status of individual tags in arrays indexed by TagID, using copy-on-write (COW) for lock-free reads.

**Performance target**: ~485ns per log call (0.76x - **24% faster than integer baseline!**).

**Rationale for hybrid COW approach**: At Mattermost scale (50K users, 8-node cluster, 10K logs/sec peak per node), pure string caching would be ~60% slower than COW approach. While absolute CPU savings are modest at this scale, the 24% performance improvement over integers provides important headroom during traffic spikes. The lock-free reads eliminate contention and ensure perfect scaling across CPU cores.

## Design Decisions

### 1. Replace Levels with Tags Completely

**Decision**: Tags will completely replace the current Level system internally, while maintaining all existing APIs for backward compatibility.

**Rationale**: 
- Eliminates dual complexity (StdFilter + CustomFilter)
- Provides unified, flexible filtering mechanism
- Removes artificial ID hierarchy constraints
- Enables better performance with simpler caching

### 2. Backward Compatibility Strategy

**Decision**: All existing APIs remain unchanged. Log levels are converted to tags internally.

**Implementation**:
```go
// Existing API (unchanged)
logger.Info("message")

// Internal conversion
logger.WithTags("info").Log("message")
```

**Default tag mappings**:
- `logger.Error()` → `WithTags("error")`
- `logger.Debug()` → `WithTags("debug")`
- etc.

### 3. Tag Filtering Logic

**Decision**: Use **ANY match** logic for tag filtering with exclusion support.

**Logic**:
1. Check exclusion tags first (fast rejection)
2. If any log tag matches any enabled tag in config → enabled
3. If log tag matches any exclusion tag → disabled

**Example**:
```go
// Config: ["debug", "auth", "!content"]
logger.WithTags("debug", "auth").Log("message")     // ✅ Enabled (has "debug")
logger.WithTags("debug", "content").Log("message")  // ❌ Disabled (has "!content")
logger.WithTags("info", "auth").Log("message")      // ✅ Enabled (has "auth")
```

### 4. Tag Attributes (Stacktrace & Color)

**Decision**: Encode attributes directly in configuration tag names, not in logging calls.

**Syntax**:
- `^` prefix = stacktrace required
- `:color` suffix = color specification
- Combined: `^error:red`

**Examples**:
```json
{
  "console": {
    "tags": ["info:cyan", "warn:yellow", "^error:red", "debug", "!content"]
  }
}
```

**Logging calls remain clean**:
```go
logger.WithTags("error", "auth").Log("message")  // Clean, no attributes
```

### 5. Exclusion Tags

**Decision**: Support exclusion tags using `!` prefix in configuration.

**Usage**: Filter out noisy debug content while keeping other debug logs.

**Example**:
```json
{
  "debug_file": {
    "tags": ["debug", "^error", "!content", "!performance"]
  }
}
```

**Result**: Shows all debug and error logs except those tagged with "content" or "performance".

### 6. Hybrid String-to-Integer Interning with COW Per-Tag Array Caching

**Decision**: Automatically intern string tags to integer IDs and cache individual tag status in copy-on-write (COW) arrays for lock-free reads (not combinations).

**Rationale**:
- Pure string caching: sort+join for cache keys = ~150ns per lookup × 5 lookups = 750ns overhead
- Combination caching (even with integers): still need key generation + sorting
- **Per-tag array caching**: No key generation, no sorting, just direct array access
- At Mattermost scale (10K logs/sec peak per node), pure strings would be 60% slower

**Implementation**:
- String tags automatically assigned sequential integer IDs (1, 2, 3, ...)
- **Cache individual tag status in arrays indexed by TagID** (not combinations!)
- API remains string-based (no developer impact)
- Reverse mapping maintained for display/formatting

**Why Per-Tag Arrays Win**:
```go
// Combination caching (rejected):
cacheKey := sort + join([5, 100, 140]) → "5,100,140" (~30ns)
result := cache.Load(cacheKey) (~20ns)
Total: ~50ns per target

// Per-tag array caching (implemented):
for _, id := range [5, 100, 140] {
    if tagIncluded[id] { return true }  // 4 tags × 2ns = 8ns
}
Total: ~8ns per target (no key generation!)
```

**Key insight**: Tag filtering uses ANY-match logic, so we can check each tag individually instead of caching combinations. Order doesn't matter, no sorting needed!

**Performance**: Interning cost ~100ns per log, but eliminates all key generation overhead (~265ns saved) = net 165ns gain.

## Legacy Code Removal and Refactoring

This section documents what needs to be removed or significantly refactored during the tag system implementation.

### Types to Remove Completely

**1. `Level` type (`level.go`)**
- **Current**: Struct representing hierarchical log levels (Debug, Info, Warn, Error, etc.)
- **Replacement**: Tags (strings internally converted to TagIDs)
- **Rationale**: Tags eliminate artificial hierarchy constraints
- **API Impact**: None - logger methods like `Debug()`, `Info()`, etc. remain unchanged as primary API

**2. `StdFilter` type (`filterstd.go`)**
- **Current**: Standard filter implementation using Level hierarchy
- **Replacement**: `TagFilter` with ANY-match logic
- **Rationale**: Tags don't need hierarchical comparison (level >= threshold)
- **Migration**: Convert StdFilter configs to equivalent tag lists

**3. `CustomFilter` type (`filtercustom.go`)**
- **Current**: Custom filter for non-standard levels
- **Replacement**: Unified `TagFilter` handles all cases
- **Rationale**: No distinction between "standard" and "custom" with tags
- **Migration**: All custom levels become tags automatically

**4. `LevelStatus` type (`levelcache.go`)**
- **Current**: Caches whether a level is enabled and requires stacktrace
- **Replacement**: `TagStatus` with same fields but for tag combinations
- **Rationale**: Same caching concept, different key type (tags vs levels)

### Types to Refactor Significantly

**1. `Filter` interface (`filter.go`)**

**Current signature:**
```go
type Filter interface {
    GetEnabledLevel(level Level) (Level, bool)
}
```

**New signature:**
```go
type Filter interface {
    IsEnabled(tags []string) bool
    RequiresStacktrace(tags []string) bool
}
```

**Changes:**
- Remove `GetEnabledLevel()` - no longer meaningful without level hierarchy
- Add `IsEnabled(tags []string) bool` - ANY-match logic for tag filtering
- Add `RequiresStacktrace(tags []string) bool` - check if any tag requires stacktrace
- Simpler interface: boolean checks instead of level resolution

**Rationale for keeping Filter interface:**
- ✅ Maintains extensibility - users can implement custom filtering logic
- ✅ Enables testing - easy to mock filter behavior
- ✅ Decouples filtering from caching - filters define logic, caches optimize it
- ✅ Future-proof - room for AND/OR/NOT combinations or other logic

**Alternative considered: Remove Filter interface entirely**
- ❌ Would hardcode tag filtering logic into TargetHost
- ❌ Loses extensibility for custom filtering strategies
- ❌ Makes testing harder (can't inject test filters)
- ❌ No performance benefit (caching eliminates filter call overhead on hot path)

**Decision: Keep Filter interface** with new signature. Filters are only called during cache rebuilds (cold path), not during log emission (hot path), so flexibility outweighs any theoretical simplification.

### Files to Remove

- `filterstd.go` - standard level filtering
- `filtercustom.go` - custom level filtering

### Files to Refactor Significantly

- `filter.go` - Update Filter interface
- `level.go` - Remove internal Level type implementation (can delete file entirely)
- `levelcache.go` - Rename to `tagcache.go`, replace LevelStatus with TagStatus
- `logr.go` - Add interning infrastructure, replace level caching with tag caching
- `targethost.go` - Replace level cache with per-tag array cache
- `logger.go` - Update log methods to use tags internally (API signatures unchanged)

### Backward Compatibility Strategy

**Keep as first-class API (NOT deprecated):**
```go
// Logger convenience methods remain unchanged as primary API
// They simply use tags internally instead of Level types
func (l *Logger) Debug(msg string) {
    l.WithTags("debug").Log(msg)  // Internal implementation
}

func (l *Logger) Info(msg string) {
    l.WithTags("info").Log(msg)
}

func (l *Logger) Warn(msg string) {
    l.WithTags("warn").Log(msg)
}

func (l *Logger) Error(msg string) {
    l.WithTags("error").Log(msg)
}
// ... etc - all level methods remain as core, supported API
```

**Rationale:**
- These methods are too commonly used to deprecate
- Users don't need to change any code
- Internal implementation changes from Level to tags (invisible to users)
- No breaking changes to public API

**Remove entirely (internal only):**
- StdFilter, CustomFilter implementations
- Level type (internal struct, not public constants)
- LevelStatus type (replace with TagStatus)
- Old Filter interface signature

### Migration Checklist

**Phase 1: Core Removal**
- [ ] Remove `StdFilter` and `CustomFilter` types
- [ ] Update `Filter` interface signature
- [ ] Implement `TagFilter` with ANY-match logic
- [ ] Replace `LevelStatus` with `TagStatus`
- [ ] Add string-to-integer interning infrastructure
- [ ] Implement COW per-tag array caching

**Phase 2: API Compatibility**
- [ ] Convert existing level methods to use tags internally
- [ ] Keep all logger method signatures unchanged (Debug, Info, Warn, Error, etc.)
- [ ] Update configuration parsing to handle both formats
- [ ] Ensure all tests pass with new implementation

**Phase 3: Cleanup**
- [ ] Remove all internal Level type implementation
- [ ] Update documentation to explain tags (show both convenience methods and WithTags)
- [ ] Add user guide showing new tag capabilities (exclusions, attributes, etc.)
- [ ] Remove legacy filter implementations entirely

### 7. Copy-on-Write (COW) Per-Tag Array Caching Strategy

**Decision**: Cache individual tag status in dynamically-growing copy-on-write (COW) arrays indexed by TagID, not tag combinations. Use atomic.Value for lock-free reads.

**Architecture**: Two-level COW caching using arrays instead of maps:

1. **Top-level cache (Logr):**
   - Arrays indexed by TagID for each target's config
   - `tagIncluded[tagID]`: is this tag included by ANY target?
   - `tagExcluded[tagID]`: is this tag excluded by ANY target?
   - Checked once per log record

2. **Per-target cache (TargetHost):**
   - Arrays indexed by TagID for this specific target
   - `tagIncluded[tagID]`: is this tag in this target's config?
   - `tagExcluded[tagID]`: is this tag excluded by this target?
   - Checked during fanout for each target

**Why Arrays Not Maps**:
- No cache key generation needed (TagID is the index)
- No sorting needed (order irrelevant with ANY-match logic)
- Direct O(1) array access: `tagIncluded[5]` vs map lookup
- Simpler and faster: ~2ns per tag check vs ~50ns for map with key gen

**Dynamic Array Growth with Copy-on-Write (COW)**:

With logging, the read-to-write ratio is extremely high (millions of log checks vs rare tag additions during config changes). This makes **copy-on-write (COW) with lock-free reads** the optimal strategy:

```go
const (
    // Initial capacity: generous allocation to avoid early growth
    initialTagCacheCapacity = 128
)

// Wrapper for immutable array pair (enables atomic pointer swap)
type tagArrays struct {
    included []bool  // Never modified after creation
    excluded []bool  // Never modified after creation
}

type TargetHost struct {
    tagArrays atomic.Value  // stores *tagArrays (COW: lock-free reads)
    growMux   sync.Mutex    // only for coordinating growth between writers
}

// Initialize with generous capacity
func newTargetHost(...) *TargetHost {
    h := &TargetHost{
        // ... other fields
    }
    h.tagArrays.Store(&tagArrays{
        included: make([]bool, initialTagCacheCapacity),
        excluded: make([]bool, initialTagCacheCapacity),
    })
    return h
}

// Grow array when new TagID appears (COW: copy → grow → atomic swap)
func (h *TargetHost) ensureCapacity(id TagID) {
    // Fast path: check without lock (lock-free read)
    current := h.tagArrays.Load().(*tagArrays)
    if int(id) < len(current.included) {
        return  // Already big enough
    }

    // Slow path: need to grow (acquire mutex for coordination)
    h.growMux.Lock()
    defer h.growMux.Unlock()

    // Double-check after acquiring lock (another goroutine may have grown it)
    current = h.tagArrays.Load().(*tagArrays)
    if int(id) < len(current.included) {
        return
    }

    // COW: Create new larger arrays (2x growth for aggressive expansion)
    newSize := max(int(id)+1, len(current.included)*2)
    newArrays := &tagArrays{
        included: make([]bool, newSize),
        excluded: make([]bool, newSize),
    }
    copy(newArrays.included, current.included)
    copy(newArrays.excluded, current.excluded)

    // Atomic swap - readers see old or new, never half-updated
    h.tagArrays.Store(newArrays)
}
```

**Copy-on-Write (COW) benefits:**
- ✅ **Lock-free reads**: Readers never block (~5ns atomic load vs 20ns RWMutex)
- ✅ **Zero read contention**: Perfect scaling with CPU cores
- ✅ **Readers never blocked by writers**: Growth happens in parallel
- ✅ **Simple memory safety**: Old arrays stay valid until all readers done
- ✅ **Saves 75ns per log** (15ns × 5 checks) = ~13% faster overall

**Growth strategy rationale:**
- **Initial capacity: 128 entries** - handles most applications without any growth
- **2x growth factor** - minimizes number of copy operations
- **Memory trade-off justified**: Copy cost (~500ns per growth) happens once per capacity level
- **Writer coordination only**: Writers coordinate via mutex, readers unaffected

**Growth pattern (with 2x factor):**
- 128 → 256 → 512 → 1,024 entries
- Reaches 1,000 tags in just **3 copy operations** (rare events)
- Each copy: ~500ns (allocate + copy ~512 entries)
- Total one-time cost: ~1,500ns across entire lifetime

**Memory efficiency**: With sequential TagID assignment and COW:
- 100 tags with 128 initial capacity: 128 × 2 arrays × 1 byte = 256 bytes per target
- 4 targets = 1,024 bytes total (~1KB)
- 1,000 tags (after growth to 1,024): 1,024 × 2 × 1 byte = 2KB per target = 8KB total
- During growth: old + new exist briefly (~1KB extra for ~1μs)
- **Negligible memory cost, massive read performance gain**

**Cache invalidation**:
- Rebuild arrays when targets added/removed or configuration changes
- Happens infrequently in production (acceptable cost)

## Implementation Architecture

### Core Data Structures

```go
// TagID is the internal integer representation of a string tag
type TagID uint32

// Tag attributes parsed from config
type TagAttributes struct {
    Name       string
    Stacktrace bool
    Color      Color
}

// TagStatus represents whether a tag combination is enabled and
// requires a stack trace (similar to current LevelStatus)
type TagStatus struct {
    Enabled    bool
    Stacktrace bool
}

// Wrapper for immutable top-level tag arrays (COW pattern)
type logrTagArrays struct {
    includedByAny []bool  // Is tag included by ANY target?
    excludedByAny []bool  // Is tag excluded by ANY target?
    needsStack    []bool  // Does ANY target need stacktrace for this tag?
}

// Logr with interning infrastructure and COW per-tag arrays
type Logr struct {
    // ... existing fields ...

    // String-to-integer interning
    stringToID   sync.Map       // map[string]TagID - intern strings to IDs
    idToString   sync.Map       // map[TagID]string - reverse lookup for display
    nextID       atomic.Uint32  // Auto-increment ID generator (sequential: 1, 2, 3, ...)

    // Top-level per-tag cache (COW: lock-free reads)
    tagArrays   atomic.Value  // stores *logrTagArrays (immutable after creation)
    rebuildMux  sync.Mutex    // only for coordinating cache rebuilds
}

// Enhanced filter interface
type Filter interface {
    IsEnabled(tags []string) bool
    RequiresStacktrace(tags []string) bool
}

// Tag filter implementation (per-target - NO caching, just configuration)
type TagFilter struct {
    includeTags map[string]TagAttributes  // "debug" -> {stacktrace: false, color: blue}
    excludeTags map[string]bool           // "content" -> true
    mux         sync.RWMutex              // Protects maps during reconfig
}

// Wrapper for immutable per-target tag arrays (COW pattern)
type tagArrays struct {
    included []bool  // Is tag included by THIS target?
    excluded []bool  // Is tag excluded by THIS target?
}

// TargetHost with COW per-tag array cache
type TargetHost struct {
    // ... existing fields ...

    // Per-target per-tag cache (COW: lock-free reads)
    tagArrays atomic.Value  // stores *tagArrays (immutable after creation)
    growMux   sync.Mutex    // only for coordinating growth between writers
}
```

### Configuration Format

```json
{
  "target-name": {
    "type": "console|file|tcp|syslog",
    "tags": ["info:cyan", "warn:yellow", "^error:red", "debug", "!content"],
    "format": "json|plain|gelf",
    "maxqueuesize": 1000
  }
}
```

### String-to-Integer Interning Implementation

**Purpose**: Automatically convert string tags to integer IDs for use as array indices in COW caches.

```go
// InternTags converts string tags to integer IDs (called at log site)
func (lgr *Logr) InternTags(tags []string) []TagID {
    ids := make([]TagID, len(tags))
    for i, tag := range tags {
        ids[i] = lgr.internTag(tag)
    }
    return ids
}

// internTag gets or creates a TagID for a string tag
func (lgr *Logr) internTag(tag string) TagID {
    // Fast path: already interned
    if id, ok := lgr.stringToID.Load(tag); ok {
        return id.(TagID)
    }

    // Slow path: first time seeing this tag
    newID := TagID(lgr.nextID.Add(1))
    if newID == 0 {
        panic("TagID overflow - 4 billion unique tags exceeded")
    }

    // Store bidirectional mapping
    lgr.stringToID.Store(tag, newID)
    lgr.idToString.Store(newID, tag)

    return newID
}

// GetTagName returns original string for a TagID (for display/formatting)
func (lgr *Logr) GetTagName(id TagID) string {
    if tag, ok := lgr.idToString.Load(id); ok {
        return tag.(string)
    }
    return fmt.Sprintf("<unknown:%d>", id)
}

// GetTagNames returns original strings for TagIDs
func (lgr *Logr) GetTagNames(ids []TagID) []string {
    tags := make([]string, len(ids))
    for i, id := range ids {
        tags[i] = lgr.GetTagName(id)
    }
    return tags
}
```

**Performance**:
- Warm path (tag already interned): ~25ns per tag (sync.Map lookup with string key)
- Cold path (first use of tag): ~50ns (add + 2 stores)
- Typical: 4 tags × 25ns = 100ns interning cost per log

### Tag Parsing Logic

**When**: ParseTag is called during configuration loading when targets are created. Tag strings from the configuration are parsed once and stored in the TagFilter's `includeTags` and `excludeTags` maps. During log emission, only fast map lookups are performed - no parsing occurs on the hot path.

```go
func ParseTag(tag string) TagAttributes {
    attrs := TagAttributes{}
    isExclusion := false

    // Check for exclusion and stacktrace prefixes in any order
    for len(tag) > 0 && (tag[0] == '!' || tag[0] == '^') {
        if tag[0] == '!' {
            isExclusion = true
            tag = tag[1:]
        } else if tag[0] == '^' {
            attrs.Stacktrace = true
            tag = tag[1:]
        }
    }

    // If this is an exclusion tag, return early (handled in different map)
    if isExclusion {
        attrs.Name = tag
        return attrs
    }

    // Check for color suffix
    if colonIndex := strings.LastIndex(tag, ":"); colonIndex != -1 {
        colorName := tag[colonIndex+1:]
        attrs.Color = ParseColor(colorName)
        tag = tag[:colonIndex]
    }

    attrs.Name = tag
    return attrs
}
```

### Filtering Implementation with Per-Tag Arrays

The filtering logic uses per-tag arrays for fast lookups without key generation:

1. **Check per-tag arrays** indexed by TagID (no key generation!)
2. **Use ANY-match logic**: return true if any tag is included and none excluded
3. **No cache misses**: arrays pre-populated from filter configuration

```go
// Top-level check across all targets (COW: lock-free reads)
func (lgr *Logr) AreTagsEnabled(tagIDs []TagID) TagStatus {
    // Lock-free read: just load pointer (~5ns atomic operation)
    arrays := lgr.tagArrays.Load().(*logrTagArrays)

    status := TagStatus{}

    // Check exclusions first (fast rejection)
    for _, id := range tagIDs {
        if int(id) < len(arrays.excludedByAny) && arrays.excludedByAny[id] {
            return status  // Excluded, return disabled
        }
    }

    // Check inclusions (ANY match = enabled)
    for _, id := range tagIDs {
        if int(id) < len(arrays.includedByAny) && arrays.includedByAny[id] {
            status.Enabled = true
            // Check if needs stacktrace
            if arrays.needsStack[id] {
                status.Stacktrace = true
            }
            break  // Found a match, done
        }
    }

    return status
}

// Per-target check (COW: lock-free reads)
func (h *TargetHost) AreTagsEnabled(tagIDs []TagID) bool {
    // Lock-free read: just load pointer (~5ns atomic operation)
    arrays := h.tagArrays.Load().(*tagArrays)

    // Check exclusions first (fast rejection)
    for _, id := range tagIDs {
        if int(id) < len(arrays.excluded) && arrays.excluded[id] {
            return false
        }
    }

    // Check inclusions (ANY match)
    for _, id := range tagIDs {
        if int(id) < len(arrays.included) && arrays.included[id] {
            return true
        }
    }

    return false
}

// Rebuild cache when configuration changes (COW: create new arrays)
func (lgr *Logr) rebuildTagCache() {
    lgr.rebuildMux.Lock()
    defer lgr.rebuildMux.Unlock()

    maxID := lgr.GetMaxTagID()

    // COW: Always create new arrays (don't modify old ones - readers may use them)
    current := lgr.tagArrays.Load().(*logrTagArrays)
    newSize := max(int(maxID)+1, initialTagCacheCapacity)

    // If current arrays exist and are smaller, grow with 2x factor
    if current != nil && len(current.includedByAny) > 0 {
        if int(maxID) >= len(current.includedByAny) {
            newSize = max(int(maxID)+1, len(current.includedByAny)*2)
        } else {
            newSize = len(current.includedByAny)
        }
    }

    // Create new arrays (start zeroed)
    newArrays := &logrTagArrays{
        includedByAny: make([]bool, newSize),
        excludedByAny: make([]bool, newSize),
        needsStack:    make([]bool, newSize),
    }

    // Populate from all targets
    lgr.tmux.RLock()
    defer lgr.tmux.RUnlock()
    for _, host := range lgr.targetHosts {
        for id := TagID(1); id <= maxID; id++ {
            tagName := lgr.GetTagName(id)
            if host.filter.IsEnabled([]string{tagName}) {
                newArrays.includedByAny[id] = true
                if host.filter.RequiresStacktrace([]string{tagName}) {
                    newArrays.needsStack[id] = true
                }
            }
        }
    }

    // Atomic swap - readers see old or new, never partial update
    lgr.tagArrays.Store(newArrays)
}

// Rebuild per-target cache (COW: create new arrays)
func (h *TargetHost) rebuildTagCache(lgr *Logr) {
    h.growMux.Lock()
    defer h.growMux.Unlock()

    maxID := lgr.GetMaxTagID()

    // COW: Always create new arrays (don't modify old ones - readers may use them)
    current := h.tagArrays.Load().(*tagArrays)
    newSize := max(int(maxID)+1, initialTagCacheCapacity)

    // If current arrays exist and are smaller, grow with 2x factor
    if current != nil && len(current.included) > 0 {
        if int(maxID) >= len(current.included) {
            newSize = max(int(maxID)+1, len(current.included)*2)
        } else {
            newSize = len(current.included)
        }
    }

    // Create new arrays (start zeroed)
    newArrays := &tagArrays{
        included: make([]bool, newSize),
        excluded: make([]bool, newSize),
    }

    // Populate from filter configuration
    for id := TagID(1); id <= maxID; id++ {
        tagName := lgr.GetTagName(id)
        newArrays.included[id] = h.filter.IsEnabled([]string{tagName})
        // Check exclusions if needed
    }

    // Atomic swap - readers see old or new, never partial update
    h.tagArrays.Store(newArrays)
}

// Optimization: Incremental cache updates for single tag additions
//
// Current approach: Full rebuild iterates through all TagIDs for all targets.
// For N targets and M tags: O(N × M) cost per rebuild.
//
// Optimization opportunity: When only adding a single new tag (not changing config),
// could do incremental update:
//
// func (lgr *Logr) addTagToCache(id TagID) {
//     current := lgr.tagArrays.Load().(*logrTagArrays)
//
//     // If array too small, still need full rebuild with growth
//     if int(id) >= len(current.includedByAny) {
//         lgr.rebuildTagCache()
//         return
//     }
//
//     // COW: copy current arrays
//     newArrays := &logrTagArrays{
//         includedByAny: append([]bool{}, current.includedByAny...),
//         excludedByAny: append([]bool{}, current.excludedByAny...),
//         needsStack:    append([]bool{}, current.needsStack...),
//     }
//
//     // Update only the new tag's entry
//     tagName := lgr.GetTagName(id)
//     lgr.tmux.RLock()
//     for _, host := range lgr.targetHosts {
//         if host.filter.IsEnabled([]string{tagName}) {
//             newArrays.includedByAny[id] = true
//             if host.filter.RequiresStacktrace([]string{tagName}) {
//                 newArrays.needsStack[id] = true
//             }
//             break
//         }
//     }
//     lgr.tmux.RUnlock()
//
//     lgr.tagArrays.Store(newArrays)
// }
//
// Trade-off analysis:
// - Full rebuild cost: O(N × M) - scales with targets and existing tags
// - Incremental cost: O(N + M) - array copy + check N targets for one tag
// - Worth it when: M > ~10 tags and config changes are rare
// - Complexity: Adds code paths, must handle array growth vs incremental update
//
// Decision: Defer to Phase 3 (Production Validation). Full rebuilds happen
// infrequently (config changes only), and cost is acceptable for most use cases.
// Profile in production to determine if incremental updates are worthwhile.

// TagFilter.IsEnabled() only used during cache rebuild, not on hot path
func (f *TagFilter) IsEnabled(logTags []string) bool {
    f.mux.RLock()
    defer f.mux.RUnlock()

    // Check exclusions first
    for _, logTag := range logTags {
        if f.excludeTags[logTag] {
            return false
        }
    }

    // Check inclusions (ANY match)
    for _, logTag := range logTags {
        if _, exists := f.includeTags[logTag]; exists {
            return true
        }
    }

    return false
}
```

## Performance Analysis

### Baseline Performance (From OPTIMIZATIONS.md Implementation)

After implementing optimizations, integer-based levels achieve:
- **Typical case (4 levels, 4 targets): ~640ns per log call**
- Breakdown:
  - Top-level cache: 4 × 40ns = 160ns
  - Per-target cache: 4 × 4 × 10ns = 160ns
  - Queue + overhead: ~320ns

### Hybrid String Tag Performance (With COW Per-Tag Array Caching)

**Performance breakdown: Hybrid with COW per-tag arrays**

| Operation | Time | Notes |
|-----------|------|-------|
| Intern 4 tags | 100ns | 4 × 25ns string→int lookups |
| Top-level cache | 13ns | 4 tags × 2ns array access + 5ns atomic.Load |
| Per-target cache (4 targets) | 52ns | 4 targets × 13ns each |
| Queue + overhead | 320ns | Same as integer system |
| **Total** | **485ns** | **0.76x - 24% FASTER than integer baseline!** |

**Typical case (4 tags, 4 targets, after warm-up):**
- Interning: 4 tags × 25ns = 100ns
- Top-level cache: 4 tags × 2ns (array) + 5ns (atomic.Load) = 13ns
- Per-target cache: 4 targets × (4 tags × 2ns + 5ns atomic.Load) = 52ns
- Queue + overhead: 320ns
- **Total: ~485ns per log call**

**Performance ratio: 485ns / 640ns = 0.76x (24% FASTER than integer baseline!)**

### Why Per-Tag Arrays Are Fastest

Per-tag arrays eliminate ALL key generation overhead:

**Pure string with combination caching (rejected):**
- Interning: 0ns (no interning)
- Cache key gen: 5 × 150ns = 750ns (sort + join strings)
- Cache lookups: 5 × 30ns = 150ns
- Total: ~900ns (without queue overhead)

**Hybrid with combination caching (rejected):**
- Interning: 100ns
- Cache key gen: 5 × 30ns = 150ns (sort + join integers)
- Cache lookups: 5 × 20ns = 100ns
- Total: ~350ns (without queue overhead)

**Hybrid with COW per-tag arrays (implemented):**
- Interning: 100ns
- Cache key gen: **0ns (eliminated!)**
- Array checks: 5 × (4 tags × 2ns) = 40ns
- atomic.Load overhead: 5 × 5ns = 25ns
- Total: ~165ns (without queue overhead)

**Key insight:** Per-tag arrays eliminate the sorting/joining bottleneck entirely. Just check arrays by TagID index!

### Performance Improvement Breakdown

Compared to integer baseline (640ns):
- Interning adds: +100ns
- COW per-tag arrays save: -255ns (vs integer combination checks + locks)
- **Net: -155ns (24% faster!)**

Compared to pure strings (1,220ns):
- COW per-tag arrays save: **735ns (60% faster!)**

**COW benefit over RWMutex approach:**
- Saves 75ns per log (15ns × 5 checks)
- 13% additional speedup from lock-free reads

### Performance at Scale (Mattermost 50K Users, 8 Nodes)

**Deployment characteristics:**
- 50K users across 8-node cluster
- Monolithic Go application (same binary on all nodes)
- Peak volume: **10,000 logs/sec per node**

**At 10K logs/sec per node (peak):**

| Approach | CPU Time | % of 1 Core | Notes |
|----------|----------|-------------|-------|
| Integer baseline | 6.4ms/sec | 0.64% | 640ns × 10K logs |
| Hybrid COW arrays | 4.85ms/sec | 0.485% | 485ns × 10K logs |
| Pure strings | 12.2ms/sec | 1.22% | 1,220ns × 10K logs |

**Per-node savings (at 10K logs/sec peak):**
- vs integers: 0.155% of 1 core (~24% faster)
- vs pure strings: 0.735% of 1 core (~60% faster)

**Cluster-wide (8 nodes):**
- vs integers: 1.24% of 1 core across cluster
- vs pure strings: 5.88% of 1 core across cluster

**Why COW matters at this scale:**
- **Percentage improvement**: 24% faster logging overhead (significant for hot paths)
- **Zero contention**: Lock-free reads scale perfectly across cores
- **Headroom**: Performance buffer for traffic spikes without adding CPU
- **Cost**: Minimal absolute cost difference, but percentage gain matters for latency-sensitive operations

### Trade-offs

**Benefits of hybrid with COW per-tag arrays:**
- ✅ **24% FASTER than integer baseline** (485ns vs 640ns per log)
- ✅ 60% faster than pure strings (485ns vs 1,220ns)
- ✅ **Lock-free reads** - zero contention, perfect scaling
- ✅ Clean string-based API (no developer impact)
- ✅ Performance headroom for traffic spikes
- ✅ Simpler than combination caching (no key generation)
- ✅ Order-independent (no sorting needed)
- ✅ Readers never blocked by writers

**Costs of hybrid:**
- ❌ ~150 lines of additional code (interning + COW array caching)
- ❌ Moderate complexity vs pure strings
- ❌ ~1KB memory for interning + ~1KB for arrays (negligible)
- ❌ Brief memory doubling during growth (~1μs, rare)
- ❌ Array rebuild on config changes (rare, acceptable)

### Why Not Pure Strings?

Pure strings without interning are unacceptable:
- 60% slower than COW hybrid approach (1,220ns vs 485ns)
- At 10K logs/sec peak: wastes 0.735% of 1 core per node
- Performance degradation impacts request latency
- No benefit to offset the performance cost

### Why Not Combination Caching?

Even with integer keys, combination caching has overhead:
- Must sort tags for consistent keys: ~10ns
- Must join/format keys: ~20ns
- Happens 5× per log = 150ns wasted
- Per-tag arrays eliminate this entirely by checking each tag individually

## Implementation Guide

This section provides concrete, step-by-step guidance for implementing the tag system.

### New API: Logger.WithTags()

**Signature:**
```go
// WithTags creates a new Logger with the specified tags.
// Tags are additive - they combine with any tags from parent loggers.
func (logger Logger) WithTags(tags ...string) Logger
```

**Behavior:**
```go
// Logger struct needs new field
type Logger struct {
    lgr    *Logr
    fields []Field
    tags   []string  // NEW: accumulated tags
}

// Implementation
func (logger Logger) WithTags(tags ...string) Logger {
    l := Logger{
        lgr:    logger.lgr,
        fields: logger.fields, // Share fields slice (immutable)
    }

    // Combine existing tags with new ones
    if len(logger.tags) > 0 || len(tags) > 0 {
        l.tags = make([]string, 0, len(logger.tags)+len(tags))
        l.tags = append(l.tags, logger.tags...)
        l.tags = append(l.tags, tags...)
    }

    return l
}
```

**Usage patterns:**
```go
// Single tag
logger := lgr.NewLogger().WithTags("debug")

// Multiple tags
logger := lgr.NewLogger().WithTags("debug", "auth", "security")

// Chaining (tags accumulate)
logger := lgr.NewLogger().
    WithTags("debug").
    WithTags("auth").
    With(Field{Key: "user", String: "bob"})
// Result: tags = ["debug", "auth"], fields = [user=bob]

// Convenience methods add their tag automatically
logger.Debug("message")  // Internally: WithTags("debug").Log("message")
```

### LogRec Structure Changes

**Current:**
```go
type LogRec struct {
    level  Level  // OLD: single level
    // ... other fields
}
```

**New:**
```go
type LogRec struct {
    tagIDs []TagID  // NEW: interned tag IDs (replaces level)
    // ... other fields
}
```

**Constructor changes:**
```go
// Old signature
func NewLogRec(lvl Level, logger Logger, msg string, fields []Field, stacktrace bool) *LogRec

// New signature
func NewLogRec(tagIDs []TagID, logger Logger, msg string, fields []Field, stacktrace bool) *LogRec {
    rec := &LogRec{
        time:    time.Now(),
        tagIDs:  tagIDs,  // Store interned IDs, not strings
        logger:  logger,
        msg:     msg,
        fields:  fields,
    }

    if stacktrace {
        rec.stackPC = make([]uintptr, DefaultMaxStackFrames)
        rec.stackCount = runtime.Callers(3, rec.stackPC)
    }

    return rec
}
```

### Configuration Parsing

**JSON/YAML format:**
```json
{
  "targets": {
    "console": {
      "type": "console",
      "tags": ["info:cyan", "warn:yellow", "^error:red", "debug", "!content"],
      "format": "plain"
    }
  }
}
```

**Parsing implementation:**
```go
// In config package
type TargetConfig struct {
    Type        string   `json:"type"`
    Tags        []string `json:"tags"`  // NEW: replaces "levels"
    Format      string   `json:"format"`
    MaxQueueSize int     `json:"maxqueuesize"`
    // ... other fields
}

// Parse each target's config
func (tc *TargetConfig) CreateFilter() Filter {
    filter := &TagFilter{
        includeTags: make(map[string]TagAttributes),
        excludeTags: make(map[string]bool),
    }

    for _, tagStr := range tc.Tags {
        attrs := ParseTag(tagStr)

        // Check for exclusion prefix (before parsing other attributes)
        if strings.HasPrefix(tagStr, "!") {
            filter.excludeTags[attrs.Name] = true
            continue
        }

        filter.includeTags[attrs.Name] = attrs
    }

    return filter
}

// ParseTag extracts attributes from config tag strings
// Called during configuration loading, NOT on hot path
func ParseTag(tag string) TagAttributes {
    attrs := TagAttributes{}
    original := tag

    // Check for exclusion prefix
    if strings.HasPrefix(tag, "!") {
        tag = tag[1:]
        attrs.Name = tag
        return attrs  // Exclusions don't have other attributes
    }

    // Check for stacktrace prefix
    if strings.HasPrefix(tag, "^") {
        attrs.Stacktrace = true
        tag = tag[1:]
    }

    // Check for color suffix
    if idx := strings.LastIndex(tag, ":"); idx != -1 {
        colorName := tag[idx+1:]
        attrs.Color = ParseColor(colorName)
        tag = tag[:idx]
    }

    attrs.Name = tag

    // Validate tag name
    if attrs.Name == "" {
        panic(fmt.Sprintf("invalid tag configuration: %q", original))
    }

    return attrs
}
```

**Backward compatibility with levels:**
```go
// Support old "levels" config for migration
type TargetConfig struct {
    Levels []string `json:"levels"`  // DEPRECATED: converted to tags
    Tags   []string `json:"tags"`    // NEW
}

// During parsing, convert levels to tags if present
if len(tc.Levels) > 0 && len(tc.Tags) == 0 {
    tc.Tags = tc.Levels  // Direct conversion: "debug" → "debug"
}
```

### Step-by-Step Implementation Order

**Phase 1: Foundation (New Types & Interfaces)**

1. **Create `tagcache.go`** (new file)
   ```go
   // Core types
   type TagID uint32

   type TagStatus struct {
       Enabled    bool
       Stacktrace bool
   }

   type TagAttributes struct {
       Name       string
       Stacktrace bool
       Color      Color
   }

   type tagArrays struct {
       included []bool
       excluded []bool
   }

   type logrTagArrays struct {
       includedByAny []bool
       excludedByAny []bool
       needsStack    []bool
   }

   const initialTagCacheCapacity = 128
   ```

2. **Update `filter.go`**
   - Change Filter interface signature
   - Document new contract
   ```go
   type Filter interface {
       IsEnabled(tags []string) bool
       RequiresStacktrace(tags []string) bool
   }
   ```

3. **Create `filtertag.go`** (new file - replaces filterstd.go and filtercustom.go)
   ```go
   type TagFilter struct {
       includeTags map[string]TagAttributes
       excludeTags map[string]bool
       mux         sync.RWMutex
   }

   func (f *TagFilter) IsEnabled(logTags []string) bool {
       // Implementation from design doc
   }

   func (f *TagFilter) RequiresStacktrace(logTags []string) bool {
       // Check if any tag has stacktrace attribute
   }

   func ParseTag(tag string) TagAttributes {
       // Implementation from above
   }
   ```

**Phase 2: Core Infrastructure (Interning & Caching)**

4. **Update `logr.go`** - Add interning infrastructure
   ```go
   type Logr struct {
       // ... existing fields ...

       // String-to-integer interning (NEW)
       stringToID  sync.Map      // map[string]TagID
       idToString  sync.Map      // map[TagID]string
       nextID      atomic.Uint32

       // Top-level COW cache (NEW)
       tagArrays   atomic.Value  // stores *logrTagArrays
       rebuildMux  sync.Mutex
   }

   // Add methods:
   func (lgr *Logr) internTag(tag string) TagID { /* ... */ }
   func (lgr *Logr) InternTags(tags []string) []TagID { /* ... */ }
   func (lgr *Logr) GetTagName(id TagID) string { /* ... */ }
   func (lgr *Logr) GetTagNames(ids []TagID) []string { /* ... */ }
   func (lgr *Logr) GetMaxTagID() TagID { /* ... */ }
   func (lgr *Logr) AreTagsEnabled(tagIDs []TagID) TagStatus { /* ... */ }
   func (lgr *Logr) rebuildTagCache() { /* ... */ }
   ```

5. **Update `targethost.go`** - Add per-target COW cache
   ```go
   type TargetHost struct {
       // ... existing fields ...

       // Per-target COW cache (NEW)
       tagArrays atomic.Value  // stores *tagArrays
       growMux   sync.Mutex
   }

   // Add methods:
   func (h *TargetHost) AreTagsEnabled(tagIDs []TagID) bool { /* ... */ }
   func (h *TargetHost) rebuildTagCache(lgr *Logr) { /* ... */ }
   func (h *TargetHost) ensureCapacity(id TagID) { /* ... */ }
   ```

**Phase 3: Data Flow (LogRec & Logger)**

6. **Update `logrec.go`**
   ```go
   type LogRec struct {
       // ... existing fields ...
       tagIDs []TagID  // CHANGED: was "level Level"
   }

   // Update constructor
   func NewLogRec(tagIDs []TagID, logger Logger, msg string, fields []Field, stacktrace bool) *LogRec

   // Add accessor for formatters
   func (rec *LogRec) Tags(lgr *Logr) []string {
       return lgr.GetTagNames(rec.tagIDs)
   }
   ```

7. **Update `logger.go`**
   ```go
   type Logger struct {
       lgr    *Logr
       fields []Field
       tags   []string  // NEW: accumulated tags
   }

   // Add new method
   func (logger Logger) WithTags(tags ...string) Logger { /* ... */ }

   // Update Log method
   func (logger Logger) Log(msg string, fields ...Field) {
       // Get tags from logger context
       tagIDs := logger.lgr.InternTags(logger.tags)

       // Check if enabled
       status := logger.lgr.AreTagsEnabled(tagIDs)
       if !status.Enabled {
           return
       }

       // Create and enqueue log record
       rec := NewLogRec(tagIDs, logger, msg, fields, status.Stacktrace)
       logger.lgr.enqueue(rec)
   }

   // Update convenience methods (NO signature changes)
   func (logger Logger) Debug(msg string, fields ...Field) {
       logger.WithTags("debug").Log(msg, fields...)
   }

   func (logger Logger) Info(msg string, fields ...Field) {
       logger.WithTags("info").Log(msg, fields...)
   }

   // ... etc for Warn, Error, Trace, Fatal, Panic
   ```

**Phase 4: Integration (Formatters & Config)**

8. **Update formatters** (formatters/*.go)
   ```go
   // In each formatter, change from:
   levelName := rec.Level().Name

   // To:
   tags := rec.Tags(lgr)
   // Format tags as needed (comma-separated, JSON array, etc.)
   ```

9. **Update config parsing** (config/*.go)
   ```go
   // Add ParseTag() integration
   // Support both "levels" and "tags" for backward compatibility
   // Convert old level configs to tag configs automatically
   ```

10. **Initialize COW caches** during Logr startup
    ```go
    func NewLogr(opts *LogrOptions) (*Logr, error) {
        lgr := &Logr{ /* ... */ }

        // Initialize with empty cache
        lgr.tagArrays.Store(&logrTagArrays{
            includedByAny: make([]bool, initialTagCacheCapacity),
            excludedByAny: make([]bool, initialTagCacheCapacity),
            needsStack:    make([]bool, initialTagCacheCapacity),
        })

        // ... rest of initialization

        return lgr, nil
    }
    ```

**Phase 5: Cleanup**

11. **Remove legacy files**
    - Delete `filterstd.go`
    - Delete `filtercustom.go`
    - Delete `level.go` (or keep minimal version for constants)

12. **Update tests**
    - Convert level-based tests to tag-based tests
    - Add new tests for tag combinations
    - Add benchmarks for interning and COW caching

### Data Flow Diagram

**Log emission flow with tags:**

```
1. User calls logger.Debug("message")
   ↓
2. Debug() calls WithTags("debug").Log("message")
   ↓
3. Log() interns tags: ["debug"] → [TagID(5)]
   Cost: ~25ns per tag (warm path)
   ↓
4. Check top-level cache: lgr.AreTagsEnabled([5])
   - Load atomic.Value (~5ns)
   - Check excluded[5] = false (2ns)
   - Check included[5] = true (2ns)
   - Check needsStack[5] = false (2ns)
   Result: TagStatus{Enabled: true, Stacktrace: false}
   Cost: ~13ns total
   ↓
5. Create LogRec with tagIDs=[5]
   ↓
6. Enqueue to Logr.in channel
   ↓
7. Logr fanout goroutine receives LogRec
   ↓
8. For each TargetHost, check per-target cache
   host.AreTagsEnabled([5])
   - Load atomic.Value (~5ns)
   - Check excluded[5] = false (2ns)
   - Check included[5] = true (2ns)
   Result: true
   Cost: ~13ns per target × 4 targets = 52ns
   ↓
9. Queue to each enabled TargetHost
   ↓
10. TargetHost goroutine formats and writes
    - GetTagNames([5]) → ["debug"] for formatter
    - Format and write to target

Total hot path: ~100ns (intern) + 13ns (top) + 52ns (targets) + 320ns (queue/overhead) = 485ns
```

### Testing Strategy

**Unit tests to add:**

1. **Tag interning** (`logr_test.go`)
   ```go
   func TestInternTags(t *testing.T) {
       // Test sequential ID assignment
       // Test bidirectional mapping
       // Test concurrent interning
   }
   ```

2. **COW caching** (`tagcache_test.go`)
   ```go
   func TestCOWArrayGrowth(t *testing.T) {
       // Test initial capacity
       // Test 2x growth
       // Test concurrent reads during growth
   }

   func TestCOWConcurrentReads(t *testing.T) {
       // Verify lock-free reads
       // Test readers never blocked by writers
   }
   ```

3. **Tag filtering** (`filtertag_test.go`)
   ```go
   func TestTagFilterAnyMatch(t *testing.T) {
       // Test ANY-match logic
       // Test exclusions override inclusions
   }

   func TestParseTag(t *testing.T) {
       // Test "^error:red"
       // Test "!content"
       // Test invalid syntax
   }
   ```

4. **Backward compatibility** (`logger_test.go`)
   ```go
   func TestLegacyLevelMethods(t *testing.T) {
       // Ensure Debug(), Info(), etc. still work
       // Verify they produce correct tags internally
   }
   ```

**Benchmarks to add:**

```go
func BenchmarkLogWithTags(b *testing.B) {
    // Target: ~485ns per log call
}

func BenchmarkInternTags(b *testing.B) {
    // Warm path: ~25ns per tag
    // Cold path: ~50ns per tag
}

func BenchmarkCOWArrayAccess(b *testing.B) {
    // Target: ~2ns per tag check
}

func BenchmarkTagFilterAnyMatch(b *testing.B) {
    // Various tag combinations
}
```

**Validation criteria:**
- ✅ All existing tests pass with tags
- ✅ Performance: 485ns ± 10% per log call (4 tags, 4 targets)
- ✅ Interning: 25ns warm path, 50ns cold path
- ✅ COW arrays: Lock-free reads, zero contention
- ✅ No race conditions detected (`go test -race`)

### Common Pitfalls to Avoid

1. **Don't modify COW arrays in place** - Always create new arrays and atomic swap
2. **Don't forget to rebuild caches** - After target add/remove or config change
3. **Don't cache tag combinations** - Cache individual tag status only
4. **Don't call ParseTag on hot path** - Only during config loading
5. **Don't use RWMutex for cache reads** - Use atomic.Value for lock-free reads
6. **Don't forget bounds checking** - Array access: `if int(id) < len(array)`
7. **Remember tag accumulation** - WithTags() chains should append, not replace

### Verification Checklist

Before considering implementation complete:

- [ ] All legacy Level-based code removed
- [ ] All tests passing (including race detector)
- [ ] Benchmarks show 485ns ± 10% target met
- [ ] Configuration parsing handles both old and new formats
- [ ] Documentation updated with tag examples
- [ ] Example applications work with new API
- [ ] Backward compatibility: all Logger convenience methods unchanged
- [ ] COW caches properly initialized at startup
- [ ] No panics on concurrent cache access

## Migration Path

### Phase 1: Core Tag System with Hybrid Interning and COW Per-Tag Arrays
- Implement string-to-integer interning infrastructure (sequential TagID assignment)
- Convert Level system to tag system internally
- Implement two-level COW per-tag array caching with atomic.Value (NOT combination caching)
- Keep all existing APIs unchanged (backward compatibility)
- Add `WithTags()` method for new functionality
- **Target performance**: ~485ns per log (0.76x - 24% faster than integer baseline!)

### Phase 2: Enhanced Configuration
- Add tag-based configuration support
- Support attribute encoding in config tags (^stacktrace, :color)
- Support exclusion tags (!tag)
- Maintain backward compatibility with level-based config
- Update documentation with examples

### Phase 3: Production Validation
- Deploy to staging environment
- Benchmark actual performance vs predicted ~485ns per log
- Compare to integer baseline (~640ns) - should be 24% faster
- Monitor CPU usage at 10K logs/sec peak (expect ~0.485% of 1 core per node)
- Validate percentage improvement: 24% faster than integers, 60% faster than pure strings
- Verify lock-free reads scale perfectly with CPU cores (zero contention)
- Test under traffic spikes to validate performance headroom
- Fine-tune if needed (should be minimal)

## Benefits

1. **Granular Control**: Enable debug logs only for specific modules
2. **Reduced Firehose**: Exclude noisy tags while keeping useful ones
3. **Flexibility**: Each target can filter differently
4. **Performance**: Efficient caching and filtering
5. **Backward Compatibility**: All existing code continues to work
6. **Simplicity**: One unified filtering system instead of multiple

## Example Use Cases

### Selective Debug Logging
```go
// Enable debug for auth module only
{
  "console": {
    "tags": ["info", "warn", "error", "debug.auth"]
  }
}

// Usage
logger.WithTags("debug", "auth").Log("Auth check")      // ✅ Shown
logger.WithTags("debug", "database").Log("SQL query")  // ❌ Hidden
```

### Cross-Cutting Concerns
```go
// Show all security-related logs regardless of level
{
  "security_file": {
    "tags": ["^security:red", "auth", "admin"]
  }
}

// Usage
logger.WithTags("debug", "security").Log("Security check")  // ✅ Shown with stacktrace
logger.WithTags("info", "auth").Log("Login attempt")        // ✅ Shown
```

### Noise Reduction
```go
// Debug logs except noisy content
{
  "debug_file": {
    "tags": ["debug", "^error", "!content", "!performance"]
  }
}

// Usage
logger.WithTags("debug", "auth").Log("Auth details")       // ✅ Shown
logger.WithTags("debug", "content").Log("Request body")    // ❌ Hidden
```

This design provides a powerful, flexible logging system that solves the "debug firehose" problem while maintaining full backward compatibility and **better-than-integer performance**.

## Summary

**Key Design Decisions:**
1. ✅ String-based API for developer experience
2. ✅ Hybrid string-to-integer interning (sequential TagIDs)
3. ✅ **Copy-on-write (COW) per-tag array caching** (NOT combination caching)
4. ✅ **Lock-free reads with atomic.Value** (zero contention)
5. ✅ Two-level COW array caching (top-level + per-target)
6. ✅ No cache key generation needed (TagID is array index)
7. ✅ Order-independent (no sorting needed)
8. ✅ Backward compatible with existing Level APIs

**Performance:**
- Target: ~485ns per log call (0.76x - **24% FASTER than integer baseline!**)
- At 10K logs/sec peak (per node): 0.485% of 1 core (vs 0.64% for integers, 1.22% for pure strings)
- Percentage improvement: 24% faster than integers, 60% faster than pure strings
- Lock-free reads: zero contention, perfect multi-core scaling
- Performance headroom for traffic spikes without adding hardware

**Implementation Complexity:**
- ~150 lines total (interning + COW per-tag arrays)
- Simpler than combination caching (no key generation logic)
- COW pattern enables lock-free reads with simple atomic.Value
- Dynamic array growth handles any number of tags
- Well-justified by performance gains and cost savings

**Result: String-based tags with COW are 24% FASTER than integer levels while providing much better flexibility and perfect read scalability!**