# LLM Proxy

[![Build Status](https://img.shields.io/badge/build-passing-brightgreen)]()
[![Version](https://img.shields.io/badge/version-1.0.0-blue)]()
[![License](https://img.shields.io/badge/license-MIT-green)]()

> **A lightweight HTTP proxy that unifies multiple LLM backends under a single API surface.**

## Why This Exists

Managing multiple LLM endpoints is painful. You might have:

- Different models for different tasks (coding vs. chat vs. analysis)
- Multiple backend servers for load balancing or redundancy
- Model-specific parameters you repeat in every request
- Clients that need a single, consistent API endpoint

Building a proxy that handles all of this is boilerplate work. This project exists so you don't have to.

## What Makes It Different

| Feature | Typical Proxy | LLM Proxy |
|---------|--------------|-----------|
| **Model routing** | Path-based only | Path + model-aware routing |
| **Default parameters** | None (client must specify every time) | Per-model defaults (temperature, top_p, etc.) |
| **Request merging** | Pass-through only | Auto-merges config + client request |
| **Configuration** | Hard-coded or env vars | YAML-based, human-readable |
| **SSE streaming** | Often broken or requires workarounds | First-class support, works out of the box |
| **Setup time** | Hours of coding | 5 minutes (build + config + run) |

**The key difference:** This isn't just a request forwarder. It's a **configuration layer** that sits between your clients and your LLM backends, handling the complexity of model routing, parameter defaults, and request merging automatically.

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
