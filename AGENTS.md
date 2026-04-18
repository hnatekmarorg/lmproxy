# Agent Guidelines for LLM Proxy

This document provides guidelines for LLM agents working on the LLM Proxy codebase.

## Table of Contents

- [Project Overview](#project-overview)
- [Codebase Structure](#codebase-structure)
- [Agent Task Guidelines](#agent-task-guidelines)
- [Code Patterns](#code-patterns)
- [Testing Guidelines](#testing-guidelines)
- [Common Tasks](#common-tasks)

---

## Project Overview

**What this project does:**
- Lightweight HTTP proxy for routing requests to multiple LLM endpoints
- Merges client requests with configured default model parameters
- Supports SSE streaming for real-time responses
- Path-based routing to different backend models

**Key technologies:**
- Go (Golang)
- YAML configuration
- HTTP/SSE handling

---

## Codebase Structure

```
.
├── main.go           # Entry point: arg parsing, config loading, server startup
├── config/
│   └── models.go     # Configuration types, YAML unmarshaling, validation
├── proxy/
│   └── config.go     # Request routing, body merging, forwarding logic
├── util/
│   └── util.go       # Map merging, deep copy, utility functions
├── config.yaml       # Example configuration
└── AGENTS.md         # This file - guidelines for AI agents
```

### File Responsibilities

| File | Responsibility | Agent Notes |
|------|---------------|-------------|
| `main.go` | Entry point, lifecycle | Don't change unless adding CLI flags |
| `config/models.go` | Config parsing, types | Safe to extend with new config fields |
| `proxy/config.go` | Core proxy logic | Be careful with routing/merging changes |
| `util/util.go` | Shared utilities | Safe to add new helper functions |

---

## Agent Task Guidelines

### When Adding New Features

1. **Read existing code first** - Understand the patterns before writing
2. **Follow existing conventions** - Match naming, style, and structure
3. **Add tests** - Include unit tests for new functionality
4. **Update documentation** - Update CONFIG.md if adding config fields

### When Fixing Bugs

1. **Reproduce the issue** - Understand the root cause before fixing
2. **Add a test case** - Ensure the bug doesn't regress
3. **Make minimal changes** - Fix only what's broken
4. **Check for side effects** - Verify no other code depends on the broken behavior

### When Refactoring

1. **Run tests first** - Establish baseline
2. **Refactor in small steps** - Commit each logical change separately
3. **Verify tests pass** - After each refactoring step
4. **Don't change behavior** - Refactoring ≠ feature changes

---

## Code Patterns

### Configuration Structure

```go
// Server configuration
type Server struct {
    Host string `yaml:"host"`
    Port int    `yaml:"port"`
}

// Endpoint configuration
type Endpoint struct {
    Host   string  `yaml:"host"`
    Models []Model `yaml:"models"`
}

// Model configuration
type Model struct {
    ID                 string            `yaml:"id"`
    Path               string            `yaml:"path"`
    Body               map[string]any    `yaml:"body"`
    ExtraBody          map[string]any    `yaml:"extra_body"`
    ChatTemplateKwargs map[string]any    `yaml:"chat_template_kwargs"`
}
```

### Request Handling Pattern

```go
// 1. Parse incoming request
// 2. Resolve target endpoint based on path
// 3. Merge config defaults with client request
// 4. Forward to backend server
// 5. Stream response back to client
```

### Error Handling Pattern

```go
// HTTP status codes to use:
// 400 - Invalid request or configuration
// 404 - Unknown route or model
// 500 - Internal proxy error
// 502 - Backend server error
```

### Map Merging Pattern (in util/util.go)

```go
// Deep merge maps for config + request body
// Client request takes precedence over defaults
// Handle nested maps recursively
```

---

## Testing Guidelines

### Test File Naming

- `main_test.go` - Tests for main.go
- `config/models_test.go` - Tests for config package
- `proxy/config_test.go` - Tests for proxy package
- `util/util_test.go` - Tests for util package

### Test Coverage Priorities

1. **Configuration loading** - Test YAML parsing and validation
2. **Request routing** - Test path resolution to endpoints
3. **Body merging** - Test config + request merging logic
4. **Error handling** - Test all error paths

### Example Test Structure

```go
func TestLoadConfig(t *testing.T) {
    // 1. Setup: Create test config file
    // 2. Execute: Load config
    // 3. Verify: Check all fields parsed correctly
    // 4. Cleanup: Remove temp file
}

func TestRouteRequest(t *testing.T) {
    // Test path resolution to correct endpoint
}

func TestMergeBody(t *testing.T) {
    // Test config defaults + client request merging
}
```

---

## Common Tasks

### Task: Add a New Configuration Field

1. Add field to the appropriate struct in `config/models.go`
2. Add YAML tag with field name
3. Add default value if needed
4. Add validation if required
5. Update CONFIG.md documentation
6. Add test case for the new field

### Task: Add a New Utility Function

1. Check if similar function exists in `util/util.go`
2. Add function with clear name and documentation
3. Add unit tests in `util/util_test.go`
4. Use in target code
5. Verify all tests pass

### Task: Fix a Routing Bug

1. Read `proxy/config.go` to understand routing logic
2. Identify the bug (path matching, endpoint resolution, etc.)
3. Write failing test case first
4. Fix the routing logic
5. Verify test passes
6. Check for similar patterns elsewhere

### Task: Add Logging Configuration Support

1. Add `Logging` struct to `config/models.go`:
   ```go
   type Logging struct {
       Level  string `yaml:"level"`
       Format string `yaml:"format"`
   }
   ```
2. Add `Logging` field to main config struct
3. Add default values (level: "info", format: "text")
4. Update proxy to use logging config
5. Update CONFIG.md with new fields

### Task: Add Timeout Configuration

1. Add `Timeout` field to main config struct in `config/models.go`
2. Add default value (30 seconds)
3. Apply timeout to HTTP client in proxy
4. Update CONFIG.md
5. Add test for timeout behavior

---

## Architecture Notes

### Request Flow

```
Client Request
    ↓
Path Resolution (proxy/config.go)
    ↓
Config Lookup (config/models.go)
    ↓
Body Merging (util/util.go)
    ↓
Backend Forwarding (proxy/config.go)
    ↓
Response Streaming (proxy/config.go)
```

### Key Design Decisions

1. **Path-based routing** - Routes matched by URL prefix
2. **Config merging** - Defaults merged with client request (client wins)
3. **SSE support** - Streaming responses forwarded as-is
4. **No auth layer** - Proxy doesn't add authentication

### Things to Be Careful About

⚠️ **Don't change the config YAML structure** without updating:
- All config structs
- Default values
- Validation logic
- CONFIG.md documentation

⚠️ **Don't break backward compatibility** - Existing config files should still work

⚠️ **Don't introduce blocking I/O** - Keep the server non-blocking for concurrency

---

## Quick Reference

### Common Commands

```bash
# Run all tests
go test -v ./...

# Run tests with coverage
go test -v -cover ./...

# Build the binary
go build -o lmproxy main.go

# Run with config
./lmproxy config.yaml
```

### Files to Read for Common Tasks

| Task | Read First |
|------|------------|
| Add config field | `config/models.go` |
| Fix routing bug | `proxy/config.go` |
| Add utility function | `util/util.go` |
| Change CLI behavior | `main.go` |
| Understand request flow | `proxy/config.go` + `main.go` |

---

## See Also

- [README.md](README.md) - User-facing documentation
- [CONFIG.md](CONFIG.md) - Configuration reference
- [config.yaml](config.yaml) - Example configuration
