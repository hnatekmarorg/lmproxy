# LM-Proxy Installer

An interactive wizard that generates `config.yaml` and systemd service files for LM-Proxy.

## Usage

### Build the installer

```bash
go build -o installer cmd/install/main.go
```

### Run the installer

```bash
./installer
```

## What the installer creates

1. **config.yaml** - Configuration file with your settings
2. **lmproxy.service** (optional) - Systemd service unit file

## Configuration steps

The installer will prompt you for:

### Server Configuration
- Listen port (default: 9090)
- Log level (debug/info/warn/error)
- Log format (text/json)

### Path Configuration
- Install directory (where files will be placed)

### Endpoint Configuration
- LLM server URL (e.g., https://api.openai.com, http://localhost:11434)
- Models to expose through the proxy
  - Model ID (used in proxy path)
  - Proxy path (e.g., /v1/chat/completions)
  - Backend model name
  - Temperature, TopP, TopK settings
  - Streaming enabled/disabled
  - Repetition penalty

### Systemd Service (optional)
- Service username
- Whether to install binary to /usr/local/bin

## Example output

```
==========================================
    LM-Proxy Quick Install Wizard
==========================================

--- Server Configuration ---
Listen port [9090]: 8080
Log level (debug/info/warn/error) [info]: debug
Log format (text/json) [text]: json

--- Path Configuration ---
Install directory (for config.yaml) [/home/user]: /opt/lmproxy

--- Endpoint Configuration ---
[Endpoint 1]
LLM server URL (e.g., https://api.openai.com): http://localhost:11434

Model configuration for this endpoint:
[Model 1]
Model ID: llama3
Proxy path [/v1/chat/completions]: 
Backend model name (or press Enter to use model ID): 
Temperature (0.0 - 2.0) [0.70]: 
Top P (0.0 - 1.0) [0.95]: 
Top K [20]: 
Enable streaming? [Y/n]: 
Repetition penalty [1.00]: 

Add another model to this endpoint? [Y/n]: N
Add another endpoint? [Y/n]: N

--- Systemd Service ---
Install binary to /usr/local/bin? [y/N]: y
Generate systemd service file? [Y/n]: 
Systemd service user (or press Enter for 'nobody'): lmproxy

Configuration Summary:
----------------------
Server Port: 8080
Log Level: debug
Log Format: json
Endpoints: 1
  1. http://localhost:11434 (1 models)
Generate systemd service file: true
Service Binary Path: /usr/local/bin/lmproxy

Generate configuration files? [Y/n]: 

✅ Installation complete!
```

## Installation steps after wizard

```bash
# 1. Build the proxy
go build -o lmproxy main.go

# 2. Install binary (if opted in)
sudo cp lmproxy /usr/local/bin/

# 3. Install systemd service
sudo install -m 644 lmproxy.service /etc/systemd/system/

# 4. Enable and start
sudo systemctl daemon-reload
sudo systemctl enable --now lmproxy

# 5. Check status
sudo systemctl status lmproxy

# 6. View logs
sudo journalctl -u lmproxy -f
```

## Non-systemd usage

If you didn't generate a systemd service, simply run:

```bash
./lmproxy config.yaml
```
