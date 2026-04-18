# LLM Proxy

[![Build Status](https://img.shields.io/badge/build-passing-brightgreen)]()
[![Version](https://img.shields.io/badge/version-1.0.0-blue)]()
[![License](https://img.shields.io/badge/license-MIT-green)]()

> **Shift the burden of sampling parameters from clients to the proxy server.**

## Why This Exists

**The problem:** Every client request needs to specify sampling parameters like `temperature`, `top_p`, `top_k`, `max_tokens`, etc. This creates:

- **Repetition** - Same parameters sent in every request
- **Drift** - Different clients using different defaults
- **Maintenance burden** - Changing a parameter means updating all clients
- **Leaky abstraction** - Clients need to know model-specific details

**The solution:** Define sampling parameters once in the proxy configuration. Clients send clean, simple requests. The proxy injects the right parameters based on which model they're using.

```yaml
# Server-side configuration (defined once)
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
# Client request (no params needed)
curl -X POST http://proxy/coding/v1/chat/completions \
  -d '{"messages": [...]}'

# Proxy automatically adds:
# {"temperature": 0.1, "top_p": 0.95, "top_k": 40, "max_tokens": 8192, ...}
```

## What Makes It Different

| Feature | Typical Proxy | LLM Proxy |
|---------|--------------|-----------|
| **Sampling params** | Client must specify every time | Defined once in server config |
| **Default parameters** | None | Per-model defaults (temperature, top_p, etc.) |
| **Request merging** | Pass-through only | Auto-merges config + client request |
| **Configuration** | Hard-coded or env vars | YAML-based, human-readable |
| **SSE streaming** | Often broken or requires workarounds | First-class support, works out of the box |
| **Setup time** | Hours of coding | 5 minutes (build + config + run) |

**The key difference:** This isn't just a request forwarder. It's a **parameter management layer** that centralizes model configuration, so clients don't need to know about temperature, top_p, or any other sampling details.

## Features

- **Multi-endpoint support** - Route to multiple backend LLM servers
- **Model-specific configuration** - Per-model defaults (temperature, top_p, etc.)
- **Request body merging** - Auto-merge client requests with configured defaults
- **SSE streaming support** - Full Server-Sent Events streaming
- **Flexible path routing** - Map custom paths to different models

## Quick Start

1. **Build the binary**

   ```bash
   go build -o lmproxy main.go
   ```

2. **Create a config file** (see [CONFIG.md](CONFIG.md) for full reference)

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

3. **Run the proxy**

   ```bash
   ./lmproxy config.yaml
   ```

## Basic Usage

Send requests to your configured model paths:

```bash
curl -X POST http://localhost:9090/my-model/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

## Configuration

See [CONFIG.md](CONFIG.md) for the complete configuration reference.

### Minimal Example

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

## License

MIT
