# LLM Proxy

[![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue)]()
[![License](https://img.shields.io/badge/license-MIT-green)]()

> **Shift the burden of sampling parameters from clients to the proxy server.**

A lightweight HTTP proxy for routing LLM requests that centralizes model configuration. Define `temperature`, `top_p`, `top_k`, `max_tokens` once in your server config—clients send clean, simple requests without any sampling parameters.

---

## Table of Contents

- [Why This Exists](#why-this-exists)
- [Features](#features)
- [Quick Start](#quick-start)
- [Usage](#usage)
- [Architecture](#architecture)
- [Project Structure](#project-structure)
- [Configuration](#configuration)
- [Deployment Guides](#deployment-guides)
- [Troubleshooting](#troubleshooting)
- [Development](#development)
- [License](#license)

---

## Why This Exists

**The problem:** Every client request needs to specify sampling parameters like `temperature`, `top_p`, `top_k`, `max_tokens`, etc. This creates:

- **Repetition** — Same parameters sent in every request
- **Drift** — Different clients using different defaults
- **Maintenance burden** — Changing a parameter means updating all clients
- **Leaky abstraction** — Clients need to know model-specific details

**The solution:** Define sampling parameters once in the proxy configuration. Clients send clean, simple requests. The proxy injects the right parameters based on which model they're using.

```yaml
# Server-side configuration (defined once)
endpoints:
  - host: "https://your-llm-server.com"
    models:
      - id: coding-model
        path: /coding
        body:
          temperature: 0.1
          top_p: 0.95
          top_k: 40
          max_tokens: 8192
```

```bash
# Client request (no params needed)
curl -X POST http://proxy/coding/v1/chat/completions \
  -d '{"messages": [{"role": "user", "content": "Hello"}]}'

# Proxy automatically adds:
# {"temperature": 0.1, "top_p": 0.95, "top_k": 40, "max_tokens": 8192, ...}
```

---

## Features

| Feature | Typical Proxy | LLM Proxy |
|---------|--------------|-----------|
| **Sampling params** | Client must specify every time | Defined once in server config |
| **Default parameters** | None | Per-model defaults (temperature, top_p, etc.) |
| **Request merging** | Pass-through only | Auto-merges config + client request |
| **Configuration** | Hard-coded or env vars | YAML-based, human-readable |
| **SSE streaming** | Often broken or requires workarounds | First-class support, works out of the box |
| **Setup time** | Hours of coding | 5 minutes (build + config + run) |
| **Multi-endpoint** | Complex routing logic | Simple YAML array configuration |

The key difference: This isn't just a request forwarder. It's a **parameter management layer** that centralizes model configuration, so clients don't need to know about temperature, top_p, or any other sampling details.

---

## Quick Start

Get up and running in 5 minutes:

### 1. Build the binary

```bash
go build -o lmproxy main.go
```

### 2. Create a config file

```yaml
# config.yaml
server:
  port: 9090

endpoints:
  - host: https://your-llm-server.com
    models:
      - id: my-model
        path: /my-model
        body:
          model: my-model-name
          temperature: 0.7
```

### 3. Run the proxy

```bash
./lmproxy config.yaml
```

That's it. Your proxy is now listening on `http://localhost:9090`.

---

## Usage

### Basic Request

Send a chat completion request to your configured model:

```bash
curl -X POST http://localhost:9090/my-model/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [
      {"role": "user", "content": "Hello, how are you?"}
    ]
  }'
```

### Streaming Request

Enable SSE streaming (if configured with `stream: true` or passed by the client):

```bash
curl -X POST http://localhost:9090/my-model/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d '{
    "messages": [{"role": "user", "content": "Tell me a story"}],
    "stream": true
  }'
```

### Overriding Parameters

Client-provided parameters take precedence over configured defaults:

```bash
curl -X POST http://localhost:9090/my-model/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [{"role": "user", "content": "Write a poem"}],
    "temperature": 0.9
  }'
# Uses configured defaults EXCEPT temperature=0.9 (client override)
```

### Multiple Models

Route to different models via different paths:

```bash
# Route to the "coding" model (with low temperature defaults)
curl http://localhost:9090/coding/v1/chat/completions -d '{"messages": [...]}'

# Route to the "creative" model (with high temperature defaults)
curl http://localhost:9090/creative/v1/chat/completions -d '{"messages": [...]}'
```

---

## Architecture

```
┌──────────┐     ┌──────────────────┐     ┌─────────────┐
│  Client   │────▶│   LLM Proxy      │────▶│  Backend    │
│ (curl/SDK)│     │  localhost:9090   │     │  LLM Server │
└──────────┘     └──────────────────┘     └─────────────┘
                        │
                        ▼
               ┌──────────────────┐
               │  Config (YAML)    │
               │  - Endpoints      │
               │  - Models         │
               │  - Parameters     │
               └──────────────────┘
```

### Request Flow

1. **Client sends request** to `http://proxy/{model-path}/v1/chat/completions`
2. **Path resolution** — Proxy matches the URL path prefix to a configured model
3. **Config lookup** — Loads the endpoint host and model defaults
4. **Body merging** — Merges configured defaults with client request (client wins on conflict)
5. **Forwarding** — Forwards the merged request to the backend LLM server
6. **Response streaming** — Streams the response back to the client (with SSE support)

### Error Flow

| Status | Meaning | Common Cause |
|--------|---------|-------------|
| `400` | Bad Request | Invalid configuration or request body |
| `404` | Not Found | Request path doesn't match any model |
| `500` | Internal Error | Proxy processing failure |
| `502` | Bad Gateway | Backend server unreachable or error |

---

## Project Structure

```
.
├── main.go              # Entry point: argument parsing, config loading, server startup
├── config.yaml          # Example configuration file
├── config/
│   └── models.go        # Configuration types, YAML unmarshaling, validation
├── proxy/
│   └── config.go        # Request routing, body merging, forwarding logic
├── util/
│   └── util.go          # Map merging, deep copy, utility functions
├── CONFIG.md            # Full configuration reference
├── AGENTS.md            # Guidelines for AI agents working on the codebase
├── SYSTEMD_SETUP.md     # Systemd service setup guide
├── deploy.sh            # Deployment script
├── lmproxy.service      # Systemd service unit file
└── Dockerfile           # Docker build file
```

---

## Configuration

See [CONFIG.md](CONFIG.md) for the complete configuration reference.

### Quick Reference

```yaml
server:
  host: "0.0.0.0"         # Bind address (default: 0.0.0.0)
  port: 9090              # Listen port (default: 8080)

logging:
  level: "info"           # debug, info, warn, error
  format: "json"          # json or text

max_request_body_size: 10485760  # 10 MB default
timeout: 30                      # Request timeout in seconds

endpoints:
  - host: "https://llm-server.example.com"  # Backend LLM server
    models:
      - id: "my-model"                      # Unique model identifier
        path: "/my-model"                   # URL path prefix
        body:                               # Default request body
          model: "my-model-name"
          temperature: 0.7
          top_p: 0.95
          max_tokens: 4096
```

---

## Deployment Guides

- **Systemd** — See [SYSTEMD_SETUP.md](SYSTEMD_SETUP.md) for running as a systemd service
- **Docker** — Build with `docker build -t lmproxy .` and run with `docker run -p 9090:9090 lmproxy config.yaml`
- **Manual** — Run directly with `./lmproxy config.yaml` or use `deploy.sh`

---

## Troubleshooting

### Proxy won't start

```bash
# Check port availability
lsof -i :9090

# Validate config file syntax
./lmproxy config.yaml --dry-run   # if supported

# Run with debug logging
./lmproxy config.yaml 2>&1 | head -50
```

### Backend connection errors

- Verify backend URL is reachable: `curl https://your-llm-server.com/health`
- Check network/firewall rules
- Ensure backend server is running

### SSE streaming issues

- Ensure `stream: true` is set in the model config
- Client must set `Accept: text/event-stream` header
- Check for proxy/buffer settings that might delay streaming

### Configuration validation errors

- Verify all required fields are present (see [CONFIG.md](CONFIG.md))
- Ensure endpoint hosts include a scheme (`https://` or `http://`)
- Model paths must start with `/`

---

## Development

### Prerequisites

- Go 1.21+
- Make (optional)

### Commands

```bash
# Build
go build -o lmproxy main.go

# Run all tests
go test -v ./...

# Run tests with coverage
go test -v -cover ./...

# Run linter (if golangci-lint is installed)
golangci-lint run ./...
```

### Extending

1. **New config field** — Add to `config/models.go` struct, update defaults and validation
2. **New proxy feature** — Add logic in `proxy/config.go`
3. **New utility** — Add to `util/util.go` with tests in `util/util_test.go`
4. **Documentation** — Update [CONFIG.md](CONFIG.md) for any config changes

See [AGENTS.md](AGENTS.md) for detailed agent guidelines.

---

## License

MIT
