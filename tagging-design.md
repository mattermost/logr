# Tagging Design for Logr

## Overview

This document outlines the design for replacing the current hierarchical log level system with a flexible tag-based filtering system. Tags provide more granular control over log output while maintaining full backward compatibility.

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
// Config: ["debug", "auth", "-content"]
logger.WithTags("debug", "auth").Log("message")     // ✅ Enabled (has "debug")
logger.WithTags("debug", "content").Log("message")  // ❌ Disabled (has "-content")
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
    "tags": ["info:cyan", "warn:yellow", "^error:red", "debug", "-content"]
  }
}
```

**Logging calls remain clean**:
```go
logger.WithTags("error", "auth").Log("message")  // Clean, no attributes
```

### 5. Exclusion Tags

**Decision**: Support exclusion tags using `-` prefix in configuration.

**Usage**: Filter out noisy debug content while keeping other debug logs.

**Example**:
```json
{
  "debug_file": {
    "tags": ["debug", "^error", "-content", "-performance"]
  }
}
```

**Result**: Shows all debug and error logs except those tagged with "content" or "performance".

### 6. Caching Strategy

**Decision**: Maintain existing caching approach with tag combinations as keys.

**Implementation**:
- Cache key: `hash(sorted_tags)` or `strings.Join(sorted_tags, ",")`
- Cache value: boolean (enabled/disabled)
- Same performance characteristics as current system
- Exclusion logic doesn't complicate caching

## Implementation Architecture

### Core Data Structures

```go
// Tag attributes parsed from config
type TagAttributes struct {
    Name       string
    Stacktrace bool
    Color      Color
}

// Enhanced filter interface
type Filter interface {
    IsEnabled(tags []string) bool
    GetAttributes(tags []string) TagAttributes
}

// Tag filter implementation
type TagFilter struct {
    includeTags map[string]TagAttributes  // "debug" -> {stacktrace: false, color: blue}
    excludeTags map[string]bool           // "content" -> true
}
```

### Configuration Format

```json
{
  "target-name": {
    "type": "console|file|tcp|syslog",
    "tags": ["info:cyan", "warn:yellow", "^error:red", "debug", "-content"],
    "format": "json|plain|gelf",
    "maxqueuesize": 1000
  }
}
```

### Tag Parsing Logic

```go
func ParseTag(tag string) TagAttributes {
    attrs := TagAttributes{}
    
    // Check for exclusion (handled separately)
    if strings.HasPrefix(tag, "-") {
        return attrs // Exclusion tags handled in different map
    }
    
    // Check for stacktrace prefix
    if strings.HasPrefix(tag, "^") {
        attrs.Stacktrace = true
        tag = tag[1:]
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

### Filtering Implementation

```go
func (f *TagFilter) IsEnabled(logTags []string) bool {
    // First check exclusions (fast rejection)
    for _, logTag := range logTags {
        if f.excludeTags[logTag] {
            return false
        }
    }
    
    // Then check inclusions
    for _, logTag := range logTags {
        if _, exists := f.includeTags[logTag]; exists {
            return true
        }
    }
    
    return false
}
```

## Migration Path

### Phase 1: Internal Conversion
- Convert Level system to tag system internally
- Keep all existing APIs unchanged
- Maintain current performance characteristics
- Add `WithTags()` method for new functionality

### Phase 2: Enhanced Configuration
- Add tag-based configuration support
- Support attribute encoding in config tags
- Support exclusion tags
- Maintain backward compatibility with level-based config

### Phase 3: Optimization
- Optimize tag-based caching
- Add advanced filtering capabilities
- Consider deprecating level-based APIs (optional)

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
    "tags": ["debug", "^error", "-content", "-performance"]
  }
}

// Usage
logger.WithTags("debug", "auth").Log("Auth details")       // ✅ Shown
logger.WithTags("debug", "content").Log("Request body")    // ❌ Hidden
```

This design provides a powerful, flexible logging system that solves the "debug firehose" problem while maintaining full backward compatibility and performance.