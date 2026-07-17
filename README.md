# LLM Proxy

> **Shift the burden of sampling parameters from clients to the proxy server.**

A lightweight HTTP proxy for routing LLM requests that centralizes model configuration. Define `temperature`, `top_p`, `top_k`, `max_tokens` once in your server config — clients send clean, simple requests without any sampling parameters.

```yaml
# Server-side: define once
endpoints:
  - host: https://your-llm-server.com
    models:
      - id: coding-model
        path: /coding
        body:
          temperature: 0.1
          top_p: 0.95
          max_tokens: 8192
```

```bash
# Client: clean request, no params needed
curl -X POST http://proxy:9090/coding/v1/chat/completions \
  -d '{"messages": [{"role": "user", "content": "Hello"}]}'
```

## Quick Start

```bash
# 1. Build
go build -o lmproxy main.go

# 2. Create config.yaml (see CONFIG.md for full reference)
cat > config.yaml << 'EOF'
server:
  port: 9090
endpoints:
  - host: https://api.openai.com
    models:
      - id: gpt
        path: /gpt
        body:
          model: gpt-4o
          temperature: 0.7
EOF

# 3. Run
./lmproxy config.yaml
```

## Features

| Feature | Description |
|---------|-------------|
| **Multi-endpoint routing** | Route requests to multiple backend LLM servers |
| **Model-specific defaults** | Per-model sampling parameters (temperature, top_p, etc.) |
| **Request body merging** | Client params override server defaults automatically |
| **SSE streaming** | Full Server-Sent Events support |
| **Path routing** | `POST /my-model/v1/chat/completions` → configured backend |
| **Body-based routing** | Clients can use standard OpenAI format with `"model"` field |
| **Model discovery** | `GET /v1/models` returns all configured models (OpenAI-compatible) |
| **Docker support** | Multi-stage Dockerfile included |
| **Kubernetes** | Helm chart in `charts/lm-proxy/` |
| **Interactive setup** | Install wizard: `go run cmd/install/main.go` |

## Usage

### Path-based routing (simple)

```bash
curl -X POST http://localhost:9090/coding/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"messages": [{"role": "user", "content": "Hello"}]}'
```

### Body-based routing (OpenAI SDK compatible)

```bash
curl -X POST http://localhost:9090/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "coding-model",
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

### Model discovery

```bash
curl http://localhost:9090/v1/models
```

## Deployment

### Docker

```bash
docker build -t lmproxy .
docker run -p 9090:9090 -v ./config.yaml:/bin/config.yaml lmproxy /bin/config.yaml
```

### Systemd

See [SYSTEMD_SETUP.md](SYSTEMD_SETUP.md) for production service setup with:
- Auto-restart on crash
- Security hardening (no new privileges, protected paths)
- Journal logging
- Resource limits

### Kubernetes

```bash
helm upgrade --install lm-proxy charts/lm-proxy/ \
  --set ingress.host=lm-proxy.example.com
```

## Documentation

| Document | Audience |
|----------|----------|
| [README.md](README.md) | Users getting started |
| [CONFIG.md](CONFIG.md) | Configuration reference |
| [SYSTEMD_SETUP.md](SYSTEMD_SETUP.md) | Production deployment |
| [AGENTS.md](AGENTS.md) | AI agents contributing to this project |
| [cmd/install/README.md](cmd/install/README.md) | Interactive install wizard |

## How It Works

```
Client Request
    ↓
Path Resolution — matches URL prefix to model config
    ↓ (fallback)
Body-based Resolution — reads "model" field from POST body
    ↓
Body Merging — merges server defaults + client request (client wins)
    ↓
Backend Forwarding — proxies to LLM server
    ↓
Response Streaming — SSE or JSON back to client
```

## License

MIT
