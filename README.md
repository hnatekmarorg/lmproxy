# LLM Proxy

A lightweight HTTP proxy server for routing requests to multiple LLM endpoints with configurable model parameters.

## Features

- **Multi-endpoint support**: Route requests to multiple backend LLM servers
- **Model-specific configuration**: Define default parameters (temperature, top_p, etc.) per model
- **Request body merging**: Automatically merge client requests with configured defaults
- **SSE streaming support**: Full support for Server-Sent Events streaming
- **Flexible path routing**: Map custom paths to different models

## Installation

```bash
# Build from source
go build -o lmproxy main.go
```

### Docker

```bash
# Build the Docker image
docker build -t lmproxy .

# Run the container
docker run -v /path/to/config.yaml:/config.yaml lmproxy /config.yaml
```

## Usage

```bash
./lmproxy config.yaml
```

The proxy requires a configuration file path as a command-line argument.

## Configuration

The proxy uses a YAML configuration file. Example:

```yaml
server:
  host: 0.0.0.0  # Optional, defaults to 0.0.0.0
  port: 9090     # Optional, defaults to 8080

endpoints:
  - host: https://your-llm-server.com
    models:
      - id: qwen-coding-thinking
        path: /qwen-coding-thinking
        body:
          model: Qwen/Qwen3.5-122B-A10B-FP8
          temperature: 0.6
          top_p: 0.95
          top_k: 20
          stream: true
        chat_template_kwargs:
          enable_thinking: true
          min_p: 0.0

      - id: qwen-coding-fast
        path: /qwen-coding-fast
        body:
          model: Qwen/Qwen3.5-122B-A10B-FP8
          temperature: 0.7
          top_p: 0.8
          stream: true
        chat_template_kwargs:
          enable_thinking: false
```

### Configuration Options

#### Server
| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| host | string | No | `0.0.0.0` | Server bind address |
| port | int | No | `8080` | Server port |

#### Endpoint
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| host | string | Yes | Backend server URL |
| models | array | Yes | List of model configurations |

#### Model
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | string | Yes | Unique model identifier |
| path | string | Yes | URL path prefix for this model |
| body | object | No | Default request body parameters |
| extra_body | object | No | Additional body parameters |
| chat_template_kwargs | object | No | Chat template arguments |

## Request Flow

1. Client sends request to proxy (e.g., `POST /qwen-coding-thinking/v1/chat/completions`)
2. Proxy resolves the target endpoint based on the path
3. Model configuration is merged with client request body
4. Request is forwarded to the backend server
5. Response (including SSE streams) is forwarded back to the client

## API Endpoints

All requests are routed based on the configured model paths:

```
POST /{model-path}/v1/chat/completions
POST /{model-path}/v1/completions
POST /{model-path}/generate
```

If no specific path is provided in the request, the proxy defaults to `/v1/chat/completions`.

## Project Structure

```
.
├── main.go           # Entry point
├── config/
│   └── models.go     # Configuration types and loader
├── proxy/
│   └── config.go     # Proxy logic and request handling
├── util/
│   └── util.go       # Utility functions
├── config.yaml       # Example configuration
└── lmproxy           # Compiled binary
```

## License

MIT
