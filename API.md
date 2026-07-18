# API Reference

Complete API documentation for the LLM Proxy server.

---

## Table of Contents

- [Request Lifecycle](#request-lifecycle)
- [Endpoints](#endpoints)
- [Request Body](#request-body)
- [Parameter Merging](#parameter-merging)
- [Response Formats](#response-formats)
- [Streaming (SSE)](#streaming-sse)
- [Error Handling](#error-handling)
- [Examples](#examples)

---

## Request Lifecycle

When a client sends a request to the proxy, the following happens:

1. **Parse** — Incoming request URL, headers, and body are parsed
2. **Route** — The URL path is matched against configured model paths
3. **Load Config** — Model-specific defaults are loaded from the YAML config
4. **Merge** — Config defaults are merged with the client request body (client wins on conflicts)
5. **Forward** — The merged request is forwarded to the backend LLM server
6. **Stream/Respond** — The backend response is streamed back verbatim to the client

```
Client Request
    │
    ▼
Path Resolution: /coding/v1/chat/completions → model "coding"
    │
    ▼
Config Lookup: {temperature: 0.1, top_p: 0.95, ...}
    │
    ▼
Body Merging: defaults + client body (client wins)
    │
    ▼
Forward to: https://backend-server/coding/v1/chat/completions
    │
    ▼
Stream Response back to client
```

---

## Endpoints

### Chat Completions

```
POST /{model-path}/v1/chat/completions
```

This is the primary endpoint. Replace `{model-path}` with the path configured for your model (e.g., `/coding`, `/creative`, `/my-model`).

**Example:**

```bash
curl -X POST http://localhost:9090/my-model/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"messages": [{"role": "user", "content": "Hello"}]}'
```

**Note:** The proxy does not add any endpoints of its own. All routes are dynamic based on configured models. Requests to paths not matching any model configuration will return a `404` error.

---

## Request Body

The request body uses the standard OpenAI-compatible chat completions format. All standard fields are supported:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `messages` | array | Yes | Array of message objects with `role` and `content` |
| `model` | string | No | Model identifier (usually set in config defaults) |
| `temperature` | number | No | Sampling temperature (0.0–2.0) |
| `top_p` | number | No | Nucleus sampling parameter (0.0–1.0) |
| `top_k` | integer | No | Top-k sampling parameter |
| `max_tokens` | integer | No | Maximum tokens in the response |
| `stream` | boolean | No | Enable SSE streaming |
| `stop` | array/string | No | Stop sequences |
| `frequency_penalty` | number | No | Frequency penalty (-2.0–2.0) |
| `presence_penalty` | number | No | Presence penalty (-2.0–2.0) |

**Example with all fields:**

```json
{
  "messages": [
    {"role": "system", "content": "You are a helpful assistant."},
    {"role": "user", "content": "Write code to sort an array"}
  ],
  "temperature": 0.2,
  "top_p": 0.95,
  "max_tokens": 2048,
  "stream": true,
  "stop": ["\n\n"]
}
```

---

## Parameter Merging

The proxy merges configured defaults with client-provided parameters using a **deep merge** strategy:

### Merge Rules

1. **Client values win** — If the client provides a parameter, it overrides the configured default
2. **Nested objects** — Merged recursively, not replaced
3. **Arrays** — Replaced entirely by the client value (not merged)
4. **Missing values** — Configured defaults fill in gaps

### Examples

**Config defaults:**
```yaml
body:
  temperature: 0.7
  top_p: 0.95
  max_tokens: 4096
  stream: false
```

**Client request:**
```json
{
  "messages": [{"role": "user", "content": "Hello"}],
  "temperature": 0.3,
  "stream": true
}
```

**Merged result (sent to backend):**
```json
{
  "messages": [{"role": "user", "content": "Hello"}],
  "temperature": 0.3,       ← client wins
  "top_p": 0.95,            ← from config
  "max_tokens": 4096,       ← from config
  "stream": true            ← client wins
}
```

**Config only (no client overrides):**
```json
{
  "messages": [{"role": "user", "content": "Hello"}],
  "temperature": 0.7,       ← from config
  "top_p": 0.95,            ← from config
  "max_tokens": 4096,       ← from config
  "stream": false           ← from config
}
```

---

## Response Formats

### Non-Streaming Response

When `stream` is `false` or not set, the response is a standard JSON object:

```json
{
  "id": "chatcmpl-abc123",
  "object": "chat.completion",
  "created": 1677858242,
  "model": "my-model-name",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "Hello! How can I help you today?"
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 10,
    "completion_tokens": 10,
    "total_tokens": 20
  }
}
```

### Streaming Response (SSE)

When `stream` is `true`, the response uses Server-Sent Events (SSE):

```
data: {"id":"chatcmpl-abc123","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}

data: {"id":"chatcmpl-abc123","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}

data: {"id":"chatcmpl-abc123","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"!"},"finish_reason":null}]}

data: [DONE]
```

The proxy passes SSE responses through **without modification** — it does not buffer, cache, or transform SSE data. This ensures compatibility with all OpenAI-compatible clients.

---

## Error Handling

### Error Response Format

All errors are returned as JSON with a consistent structure:

```json
{
  "error": {
    "code": 404,
    "message": "Not found: no model configured for path \"/unknown\"",
    "details": "Available paths: /coding, /creative"
  }
}
```

### Error Codes

| Code | Meaning | Description |
|------|---------|-------------|
| `400` | Bad Request | Invalid configuration or request body format |
| `404` | Not Found | Request path doesn't match any configured model |
| `500` | Internal Error | Proxy configuration or processing error |
| `502` | Bad Gateway | Backend server unreachable or returned an error |

### Common Errors

#### 404 — Model Not Found

```json
{
  "error": {
    "code": 404,
    "message": "Not found: no model configured for path \"/unknown\"",
    "details": "Available paths: /coding, /creative"
  }
}
```

**Fixes:**
- Check the URL path matches a configured model `path`
- Verify the config file is loaded correctly
- Check proxy logs for startup errors

#### 502 — Backend Unreachable

```json
{
  "error": {
    "code": 502,
    "message": "Bad gateway: backend server returned an error",
    "details": "Failed to connect to https://llm-server.example.com: connection refused"
  }
}
```

**Fixes:**
- Verify the backend server is running
- Check network connectivity between proxy and backend
- Ensure the backend URL is correct (including scheme)

#### 500 — Configuration Error

```json
{
  "error": {
    "code": 500,
    "message": "Internal server error: failed to process request",
    "details": "Configuration validation failed: endpoint host is required"
  }
}
```

**Fixes:**
- Validate config file YAML syntax
- Ensure all required fields are present
- Check proxy startup logs for configuration errors

---

## Examples

### Minimal Setup

```bash
# Start proxy with simple config
./lmproxy config.yaml

# Send request
curl http://localhost:9090/default/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"messages": [{"role": "user", "content": "Hello"}]}'
```

### Multiple Backends

```bash
# Route to different backends based on path
curl http://localhost:9090/qa/v1/chat/completions -d '{"messages": [...]}'
curl http://localhost:9090/code/v1/chat/completions -d '{"messages": [...]}'
curl http://localhost:9090/chat/v1/chat/completions -d '{"messages": [...]}'
```

### Override Config Per-Request

```bash
# Override temperature for a single request
curl http://localhost:9090/my-model/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [{"role": "user", "content": "Be creative"}],
    "temperature": 0.9
  }'
```

### Streaming with Custom Client

```python
import requests
import json

response = requests.post(
    "http://localhost:9090/my-model/v1/chat/completions",
    json={
        "messages": [{"role": "user", "content": "Write a story"}],
        "stream": True,
    },
    stream=True,
)

for line in response.iter_lines():
    if line:
        line = line.decode("utf-8")
        if line.startswith("data: ") and line != "data: [DONE]":
            chunk = json.loads(line[6:])
            content = chunk["choices"][0]["delta"].get("content", "")
            print(content, end="", flush=True)
```
