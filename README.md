# LLM Proxy

[![Go Version](https://img.shields.io/badge/Go-1.22%2B-blue)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

> **Shift the burden of sampling parameters from clients to the proxy server.**

A lightweight HTTP proxy for routing LLM requests that centralizes model configuration. Define `temperature`, `top_p`, `top_k`, `max_tokens` once in your server config — clients send clean, simple requests without any sampling parameters.

---

## Table of Contents

- [Problem & Solution](#problem--solution)
- [Features](#features)
- [Quick Start](#quick-start)
- [Usage](#usage)
- [Configuration](#configuration)
- [API Reference](#api-reference)
- [Deployment](#deployment)
- [Development](#development)
- [FAQ / Troubleshooting](#faq--troubleshooting)
- [License](#license)

---

## Problem & Solution

### The Problem

Every client request to an LLM needs sampling parameters like `temperature`, `top_p`, `top_k`, `max_tokens`, etc. This creates:

| Pain Point | Impact |
|---|---|
| **Repetition** | Same parameters sent in every request from every client |
| **Drift** | Different clients using different defaults for the same model |
| **Maintenance burden** | Changing a parameter means updating all clients |
| **Leaky abstraction** | Clients need to know model-specific details |

### The Solution

Define sampling parameters **once** in the proxy configuration. Clients send clean, simple requests — the proxy injects the right parameters based on the model being used.

```yaml
# Define once in the proxy config (server-side)
models:
  - id: coding-model
    path: /coding
    body:
      temperature: 0.1      # Focused, deterministic
      top_p: 0.95
      top_k: 40
      max_tokens: 8192
```

```bash
# Client sends only messages — no sampling parameters needed
curl -X POST http://proxy/coding/v1/chat/completions \
  -d '{"messages": [{"role": "user", "content": "Hello"}]}'

# Proxy automatically merges:
# {"temperature": 0.1, "top_p": 0.95, "top_k": 40, "max_tokens": 8192, ...}
```

### Comparison

| Feature | Typical Proxy | LLM Proxy |
|---|---|---|
| **Sampling params** | Client must specify every time | Defined once in server config |
| **Default parameters** | None | Per-model defaults (temperature, top_p, etc.) |
|| **Request merging** | Pass-through only | Auto-merges config defaults + client request |
|| **Model discovery** | Manual URL configuration | `/v1/models` endpoint (OpenAI-compatible) |
| **Configuration** | Hard-coded or env vars | YAML-based, human-readable |
| **SSE streaming** | Often broken or requires workarounds | First-class support, works out of the box |
| **Setup time** | Hours of coding | ~5 minutes (build + config + run) |

This is **not** just a request forwarder — it's a **parameter management layer** that centralizes model configuration, so clients don't need to know about temperature, top_p, or any other sampling details.

---

## Features

- **Multi-endpoint support** — Route to multiple backend LLM servers from a single proxy
- **Model-specific configuration** — Per-model defaults for all sampling parameters
- **Request body merging** — Auto-merge client requests with configured defaults (client values take precedence)
- **SSE streaming** — Full Server-Sent Events support for real-time responses
- **Flexible path routing** — Map custom URL paths to different models and endpoints
- **Body-based routing** — Route requests by model ID (no path prefix needed)
- **YAML configuration** — Simple, human-readable, version-controllable config
- **Structured logging** — JSON or text output with configurable log levels
- **Graceful shutdown** — Handles in-flight requests on SIGINT/SIGTERM

---
>>>>>>> 6e46fa8 (docs: rewrite documentation suite for clarity and completeness)

## Quick Start

### Prerequisites

- Go 1.22+ (for building from source)
- Or Docker (for running via container — *coming soon*)

### 1. Build

```bash
git clone https://github.com/hnatekmarorg/lmproxy.git
cd lmproxy
go build -o lmproxy main.go
```

### 2. Create a Config File

Create `config.yaml`:

```yaml
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

### 3. Run

```bash
./lmproxy config.yaml
```

Your proxy is now listening at `http://localhost:9090`.

---

## Usage

### Basic Request

Send a chat completion request without any sampling parameters:

```bash
curl -X POST http://localhost:9090/my-model/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

### Model Discovery

List all available models:

```bash
curl http://localhost:9090/v1/models
```

### Body-Based Routing

For pathless models, send requests directly to `/v1/chat/completions` with the model ID in the body:

```bash
curl -X POST http://localhost:9090/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "my-model",
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

This is compatible with the OpenAI API format, so you can use any OpenAI SDK directly.

### Overriding Defaults

Client-provided parameters override the configured defaults:

```bash
curl -X POST http://localhost:9090/my-model/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [{"role": "user", "content": "Hello!"}],
    "temperature": 0.9
  }'
```

The proxy merges `{"temperature": 0.9}` on top of the model defaults.

### Streaming (SSE)

```bash
curl -X POST http://localhost:9090/my-model/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [{"role": "user", "content": "Tell me a story"}],
    "stream": true
  }'
```

Responses are streamed back as Server-Sent Events.

---

## Configuration

See the full [Configuration Reference](CONFIG.md) for detailed documentation on all options.

### Minimal Configuration

```yaml
server:
  port: 9090

endpoints:
  - host: https://your-llm-server.com
    models:
      - id: my-model
        path: /my-model
        body:
          model: my-model-name
```

### Configuration Sections

| Section | Description | Required |
|---|---|---|
| `server` | Proxy bind address, port, timeouts | Yes |
| `endpoints[]` | Backend LLM server definitions | Yes |
| `endpoints[].models[]` | Per-model configuration | Yes |
| `logging` | Log level and format | No |
| `max_request_body_size` | Max request body size in bytes | No |
| `timeout` | Request timeout in seconds | No |

See [CONFIG.md](CONFIG.md) for the complete reference including validation rules, default values, and error responses.

---

## API Reference

### Endpoints

The proxy exposes one catch-all endpoint:

#### `POST /{model-path}/v1/chat/completions`

Routes a chat completion request to the model matching `{model-path}`.

**Path Parameters:**
- `model-path` — The `path` configured for the target model (e.g., `/my-model`)

**Request Body:**
Standard OpenAI-compatible chat completion body. Sampling parameters are optional — defaults from the config are merged in automatically.

**Response:**
Streaming (if `stream: true` in request or config) or non-streaming JSON, forwarded from the backend.

### Error Responses

| Code | Name | Description |
|---|---|---|
| `400` | Bad Request | Invalid JSON, missing required fields, invalid config |
| `404` | Not Found | Unknown route or model path |
| `500` | Internal Server Error | Proxy processing error |
| `502` | Bad Gateway | Backend server unreachable or error |

Example error:

```json
{
  "error": {
    "code": 404,
    "message": "No model found for path: /unknown-model",
    "details": "Configured paths: /my-model, /other-model"
  }
}
```

---

## Deployment

### Systemd Service (Linux)

A systemd service unit is included for production deployments:

```bash
# Automated deployment
./deploy.sh

# Or manual installation
sudo cp lmproxy.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable lmproxy
sudo systemctl start lmproxy
```

See [SYSTEMD_SETUP.md](SYSTEMD_SETUP.md) for:
- Service management commands
- Resource limits configuration
- Security hardening
- Update and uninstall procedures

### Logging

```bash
# View proxy logs
journalctl -u lmproxy -f

# Check service status
systemctl status lmproxy
```

---

## Development

### Project Structure

```
.
├── main.go              # Entry point: arg parsing, config loading, server startup
├── config/
│   └── models.go        # Configuration types, YAML unmarshaling, validation
├── proxy/
│   └── config.go        # Request routing, body merging, forwarding logic
├── util/
│   └── util.go          # Map merging, deep copy, utility functions
├── config.yaml          # Example configuration
├── AGENTS.md            # Guidelines for AI-assisted development
├── CONFIG.md            # Configuration reference
├── SYSTEMD_SETUP.md     # Systemd deployment guide
└── deploy.sh            # Automated deployment script
```

### Running Tests

```bash
go test -v ./...
```

### Building

```bash
go build -o lmproxy main.go
```

For development guidelines, see [AGENTS.md](AGENTS.md).

---

## FAQ / Troubleshooting

### "connection refused" when starting

Ensure the configured port is available:
```bash
sudo lsof -i :9090
```

### "permission denied" errors

Ports below 1024 require root privileges. Either use a higher port or run with `sudo`.

### Proxy starts but requests fail with 502

The backend LLM server may be unreachable. Verify:
- The backend server is running
- The `host` URL in your config is correct and reachable
- Network/firewall rules allow outbound connections

### Config changes not taking effect

The config file is read at startup. Restart the proxy to apply changes:
```bash
systemctl restart lmproxy
```

### High memory usage

Check your `max_tokens` and `max_request_body_size` settings. Long generations with large context windows consume more memory.

---

## License

MIT — see [LICENSE](LICENSE) for details.
