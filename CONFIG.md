# Configuration Reference

Complete configuration documentation for the LLM Proxy server.

> **Note**: For an overview of the proxy and its features, see [README.md](README.md).

---

## Table of Contents

1. [Configuration File Structure](#configuration-file-structure)
2. [Server Configuration](#server-configuration)
3. [Endpoint Configuration](#endpoint-configuration)
4. [Model Configuration](#model-configuration)
5. [Complete Configuration Example](#complete-configuration-example)
6. [Error Response Codes](#error-response-codes)
7. [Validation Rules](#validation-rules)

---

## Configuration File Structure

The proxy uses a YAML configuration file with three main sections:

```yaml
server:
  # Server binding configuration
  host: "0.0.0.0"
  port: 9090

endpoints:
  # Array of backend endpoint configurations
  - host: "https://your-llm-server.com"
    models:
      # Array of model configurations per endpoint
      - id: "model-id"
        path: "/model-path"
        body: {}
        chat_template_kwargs: {}

logging:
  # Logging configuration (optional)
  level: "info"
  format: "json"

max_request_body_size: 10485760
timeout: 30
```

---

## Server Configuration

The `server` section configures the proxy's HTTP server binding.

### Server Configuration Table

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `host` | string | No | `"0.0.0.0"` | Server bind address. Accepts IPv4, IPv6, or hostname. |
| `port` | integer | No | `8080` | Server port number (1-65535). |

### Server Configuration Example

```yaml
server:
  host: "0.0.0.0"  # Bind to all network interfaces (IPv4)
  port: 9090       # Listen on port 9090
```

### Field Details

#### `host`
- **Type**: `string`
- **Required**: No
- **Default**: `"0.0.0.0"`
- **Description**: The network address the proxy server binds to.
  - `"0.0.0.0"` - Bind to all IPv4 interfaces (default)
  - `"127.0.0.1"` - Bind to localhost only
  - `"::"` - Bind to all IPv6 interfaces
  - `"localhost"` - Bind to localhost
  - Specific IP: `"192.168.1.100"`

#### `port`
- **Type**: `integer`
- **Required**: No
- **Default**: `8080`
- **Description**: The TCP port number for the proxy server.
- **Valid Range**: 1-65535
- **Note**: Ports below 1024 require root/admin privileges on Unix systems.

---

## Endpoint Configuration

The `endpoints` section defines backend LLM servers that the proxy routes requests to.

### Endpoint Configuration Table

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `host` | string | **Yes** | - | Backend server base URL (must include scheme) |
| `models` | array | **Yes** | - | List of model configurations for this endpoint |

### Endpoint Configuration Example

```yaml
endpoints:
  - host: "https://llm-server-1.example.com"  # Primary LLM server
    models:
      - id: "coding-model"
        path: "/coding"
        body:
          model: "Qwen/Qwen3.5-122B-A10B-FP8"
          temperature: 0.6

  - host: "http://localhost:8080"  # Local server
    models:
      - id: "local-model"
        path: "/local"
```

### Field Details

#### `host`
- **Type**: `string`
- **Required**: **Yes**
- **Description**: The base URL of the backend LLM server.
- **Scheme Requirements**: Must include a valid URL scheme:
  - `https://` - Secure HTTP (recommended for production)
  - `http://` - HTTP (for local development)
- **Format**: Valid URL with scheme, hostname, and optional port
  - ✅ `"https://api.example.com"`
  - ✅ `"http://localhost:8080"`
  - ❌ `"api.example.com"` (missing scheme)

#### `models`
- **Type**: `array`
- **Required**: **Yes**
- **Description**: Array of model configuration objects for this endpoint.
- **Minimum**: At least one model must be defined per endpoint.
- **Structure**: See [Model Configuration](#model-configuration) below.

---

## Model Configuration

The `models` array within each endpoint defines individual model routing configurations.

### Model Configuration Table

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `id` | string | **Yes** | - | Unique identifier for this model configuration |
| `path` | string | **Yes** | - | URL path prefix for routing requests to this model |
| `body` | object | No | `{}` | Default request body parameters |
| `extra_body` | object | No | `{}` | Additional request body parameters |
| `chat_template_kwargs` | object | No | `{}` | Chat template rendering arguments |

### Model Configuration Example

```yaml
models:
  - id: "qwen-coding-thinking"              # Unique model identifier
    path: "/qwen-coding-thinking"           # URL path for routing
    body:                                   # Default request body
      model: "Qwen/Qwen3.5-122B-A10B-FP8"
      temperature: 0.6
      top_p: 0.95
      top_k: 20
      stream: true
      max_tokens: 8192
    extra_body:                             # Additional parameters
      presence_penalty: 0.1
    chat_template_kwargs:                   # Template rendering args
      enable_thinking: true
      min_p: 0.0
```

### Field Details

#### `id`
- **Type**: `string`
- **Required**: **Yes**
- **Description**: A unique identifier for this model configuration within the endpoint.
- **Uniqueness**: Must be unique across all models in the configuration.
- **Usage**: Used for logging, metrics, and internal routing references.

#### `path`
- **Type**: `string`
- **Required**: **Yes**
- **Description**: The URL path prefix used to route requests to this model.
- **Format**: Must start with `/`
- **Examples**:
  - `/qwen-coding`
  - `/llama-chat`
  - `/generate`
- **Routing**: Requests to `POST {path}/v1/chat/completions` are routed to this model.

#### `body`
- **Type**: `object` (key-value pairs)
- **Required**: No
- **Default**: Empty object `{}`
- **Description**: Default parameters included in the proxied request body.
- **Merging Behavior**: Client request parameters override these defaults.
- **Common Fields**:
  - `model` (string): The model name sent to the backend
  - `temperature` (float): Sampling temperature (0.0-2.0)
  - `top_p` (float): Nucleus sampling parameter (0.0-1.0)
  - `top_k` (integer): Top-k sampling parameter
  - `max_tokens` (integer): Maximum response tokens
  - `stream` (boolean): Enable SSE streaming
  - `stop` (array): Stop sequences
  - `frequency_penalty` (float): Frequency penalty
  - `presence_penalty` (float): Presence penalty

#### `extra_body`
- **Type**: `object` (key-value pairs)
- **Required**: No
- **Default**: Empty object `{}`
- **Description**: Additional parameters merged into the request body.
- **Use Case**: Custom backend-specific parameters not in the standard body.
- **Merging**: Merged after `body`, before client request parameters.

#### `chat_template_kwargs`
- **Type**: `object` (key-value pairs)
- **Required**: No
- **Default**: Empty object `{}`
- **Description**: Arguments passed to the chat template renderer.
- **Use Case**: Template-specific parameters like `enable_thinking`, `min_p`, etc.
- **Note**: These are typically processed by the proxy before forwarding.

---

## Complete Configuration Example

Below is a comprehensive example with all configuration options and inline comments:

```yaml
# =============================================================================
# LLM Proxy Configuration
# =============================================================================

# -----------------------------------------------------------------------------
# Server Configuration
# -----------------------------------------------------------------------------
server:
  host: "0.0.0.0"  # Bind address: 0.0.0.0 (all interfaces), 127.0.0.1 (localhost)
  port: 9090       # Listening port: 8080 default, 9090 in this example

# -----------------------------------------------------------------------------
# Logging Configuration (Optional)
# -----------------------------------------------------------------------------
logging:
  level: "info"    # Log level: "debug", "info", "warn", "error"
  format: "json"   # Output format: "json" (structured) or "text" (human-readable)

# -----------------------------------------------------------------------------
# Request Size Limit (Optional)
# -----------------------------------------------------------------------------
max_request_body_size: 10485760  # Maximum request body size in bytes (10MB default)

# -----------------------------------------------------------------------------
# Timeout Configuration (Optional)
# -----------------------------------------------------------------------------
timeout: 30  # Request timeout in seconds (30s default)

# -----------------------------------------------------------------------------
# Endpoint Configurations
# -----------------------------------------------------------------------------
endpoints:
  # ---------------------------------------------------------------------------
  # Primary Production Endpoint
  # ---------------------------------------------------------------------------
  - host: "https://llm-server-prod.example.com"  # Backend server URL (HTTPS required)
    models:
      # Coding Model with Thinking
      - id: "qwen-coding-thinking"               # Unique model identifier
        path: "/qwen-coding-thinking"            # Routing path prefix
        body:                                    # Default request body parameters
          model: "Qwen/Qwen3.5-122B-A10B-FP8"    # Model name for backend
          temperature: 0.6                       # Creativity level (lower = more focused)
          top_p: 0.95                            # Nucleus sampling threshold
          top_k: 20                              # Top-k sampling limit
          stream: true                           # Enable SSE streaming responses
          max_tokens: 8192                       # Maximum response length
        chat_template_kwargs:                    # Template rendering arguments
          enable_thinking: true                  # Enable model thinking mode
          min_p: 0.0                             # Minimum probability threshold

      # Coding Model (Fast)
      - id: "qwen-coding-fast"
        path: "/qwen-coding-fast"
        body:
          model: "Qwen/Qwen3.5-122B-A10B-FP8"
          temperature: 0.7                       # Slightly more creative
          top_p: 0.8
          stream: true
        chat_template_kwargs:
          enable_thinking: false                 # Disable thinking for speed

      # General Purpose Model
      - id: "qwen-general"
        path: "/qwen-general"
        body:
          model: "Qwen/Qwen2.5-72B-Instruct"
          temperature: 0.7
          top_p: 0.9
          top_k: 50
          stream: true
          max_tokens: 4096
        chat_template_kwargs:
          enable_thinking: false

  # ---------------------------------------------------------------------------
  # Local Development Endpoint
  # ---------------------------------------------------------------------------
  - host: "http://localhost:8080"                # Local server (HTTP for dev)
    models:
      - id: "local-coding"
        path: "/local-coding"
        body:
          model: "local-coding-model"
          temperature: 0.5
          top_p: 0.95
          stream: true
        chat_template_kwargs:
          enable_thinking: true
```

---

## Error Response Codes

The proxy returns the following HTTP status codes:

### Error Response Table

| Code | Name | Description | Common Causes |
|------|------|-------------|---------------|
| `400` | Bad Request | Invalid request or configuration | Malformed JSON, missing required fields, invalid config |
| `404` | Not Found | Unknown route or model | Path not configured, model not found |
| `500` | Internal Server Error | Proxy processing error | Configuration error, internal failure |
| `502` | Bad Gateway | Backend server error | Backend unreachable, backend returned error |

### Error Response Details

#### 400 Bad Request

Returned when the client request or proxy configuration is invalid.

```json
{
  "error": {
    "code": 400,
    "message": "Invalid request body: missing required field 'messages'",
    "details": "The 'messages' field is required for chat completions"
  }
}
```

**Common Causes:**
- Invalid JSON in request body
- Missing required fields (`messages`, `model`)
- Invalid configuration file format
- Missing required config fields (e.g., `host`, `models`)

#### 404 Not Found

Returned when the requested route or model is not configured.

```json
{
  "error": {
    "code": 404,
    "message": "No model found for path: /unknown-model",
    "details": "Configured paths: /qwen-coding-thinking, /qwen-coding-fast"
  }
}
```

**Common Causes:**
- Request to unconfigured path
- Model path typo in request URL
- Endpoint with no models defined

#### 500 Internal Server Error

Returned when the proxy encounters an internal processing error.

```json
{
  "error": {
    "code": 500,
    "message": "Internal server error: failed to process request",
    "details": "Configuration validation failed: endpoint host is required"
  }
}
```

**Common Causes:**
- Invalid configuration file
- Missing required configuration fields
- Internal processing failure
- Type conversion errors

#### 502 Bad Gateway

Returned when the backend server is unreachable or returns an error.

```json
{
  "error": {
    "code": 502,
    "message": "Bad gateway: backend server returned an error",
    "details": "Failed to connect to https://llm-server.example.com: connection refused"
  }
}
```

**Common Causes:**
- Backend server is down
- Network connectivity issues
- Backend returned invalid response
- Backend timeout exceeded

---

## Validation Rules

### Required Fields

| Field | Location | Validation |
|-------|----------|------------|
| `host` | Endpoint | Must be present and non-empty |
| `models` | Endpoint | Must be present with at least one model |
| `id` | Model | Must be present and unique |
| `path` | Model | Must be present and start with `/` |

### URL Scheme Requirements

Endpoint `host` URLs must include a valid scheme:

| Scheme | Valid | Use Case |
|--------|-------|----------|
| `https://` | ✅ | Production (recommended) |
| `http://` | ✅ | Local development |
| (none) | ❌ | Invalid - must include scheme |
| `ftp://` | ❌ | Invalid - HTTP/HTTPS only |

### Type Validation

| Field | Expected Type | Validation |
|-------|---------------|------------|
| `server.port` | integer | Must be 1-65535 |
| `server.host` | string | Valid hostname or IP |
| `endpoints[].host` | string | Valid URL with scheme |
| `models[].path` | string | Must start with `/` |
| `logging.level` | string | One of: debug, info, warn, error |
| `logging.format` | string | One of: json, text |
| `max_request_body_size` | integer | Must be positive (> 0) |
| `timeout` | integer | Must be positive (> 0) |

### Path Validation

Model paths must follow these rules:

1. Must start with `/`
2. Cannot be empty
3. Should not contain special characters that break URLs
4. Must be unique across all endpoints

**Valid Paths:**
- `/qwen-coding`
- `/llama/chat`
- `/api/v1/model`

**Invalid Paths:**
- `qwen-coding` (missing leading `/`)
- `` (empty string)
- `http://example.com` (full URL instead of path)

### Uniqueness Constraints

| Constraint | Scope | Description |
|------------|-------|-------------|
| Model `id` | All endpoints | Each model `id` must be unique |
| Model `path` | All endpoints | Each model `path` must be unique |
| Endpoint `host` | - | No constraint (same host allowed) |

---

## Quick Reference

### Minimal Valid Configuration

```yaml
server:
  port: 8080

endpoints:
  - host: "https://api.example.com"
    models:
      - id: "default"
        path: "/default"
```

### Common Configuration Patterns

#### Local Development
```yaml
server:
  host: "127.0.0.1"
  port: 8080

endpoints:
  - host: "http://localhost:8080"
    models:
      - id: "local"
        path: "/local"
        body:
          stream: false
```

#### Production with HTTPS
```yaml
server:
  host: "0.0.0.0"
  port: 443

logging:
  level: "warn"
  format: "json"

endpoints:
  - host: "https://llm-server.example.com"
    models:
      - id: "production"
        path: "/v1"
        body:
          stream: true
```

---

## See Also

- [README.md](README.md) - Project overview and features
- [config.yaml](config.yaml) - Example configuration file
