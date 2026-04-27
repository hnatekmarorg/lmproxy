# Systemd Service Setup Guide for LLM Proxy

## Quick Start

```bash
# Deploy and start
./deploy.sh

# Or manual installation
sudo cp lmproxy.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable lmproxy
sudo systemctl start lmproxy
```

## File Locations

| Component | Path |
|-----------|------|
| Binary | `/opt/lmproxy/lmproxy` |
| Config | `/opt/lmproxy/config.yaml` |
| Service unit | `/etc/systemd/system/lmproxy.service` |
| Logs (journal) | `journalctl -u lmproxy` |

## Management Commands

```bash
# Check status
sudo systemctl status lmproxy

# View logs
sudo journalctl -u lmproxy -f
sudo journalctl -u lmproxy -n 100  # Last 100 lines

# Stop/Start/Restart
sudo systemctl stop lmproxy
sudo systemctl start lmproxy
sudo systemctl restart lmproxy

# Reload config (after editing config.yaml)
sudo systemctl restart lmproxy

# Disable auto-start
sudo systemctl disable lmproxy
```

## Configuration Options

### 1. Change Listen Port

Edit `/opt/lmproxy/config.yaml`:
```yaml
server:
  port: 9090  # Change to your desired port
```

Then restart:
```bash
sudo systemctl restart lmproxy
```

### 2. Add Environment Variables

Edit the service file:
```ini
[Service]
Environment="API_KEY=your_secret_key"
Environment="TLS_CERT_PATH=/path/to/cert.pem"
```

Reload and restart:
```bash
sudo systemctl daemon-reload
sudo systemctl restart lmproxy
```

### 3. Adjust Resource Limits

In `lmproxy.service`:
```ini
# Memory limit
MemoryMax=4G        # Increase/decrease as needed

# CPU limit (optional)
CPUQuota=200%       # Allow up to 2 cores (remove for unlimited)
```

### 4. Change Install Directory

If you want to install elsewhere (e.g., `/usr/local/bin`):

1. Edit `deploy.sh` - change `INSTALL_DIR="/opt/lmproxy"`
2. Edit `lmproxy.service` - update `WorkingDirectory`, `ExecStart`, and paths
3. Update `ReadWritePaths` if needed

## Security Considerations

### Current Security Hardening

- ✓ Runs as non-root `lmproxy` user
- ✓ No new privileges (`NoNewPrivileges=true`)
- ✓ Protected system directories (`ProtectSystem=strict`)
- ✓ Home directories protected (`ProtectHome=true`)
- ✓ Read-write access only to `/opt/lmproxy/logs`

### Additional Hardening (Optional)

For production environments, consider:

```ini
# Restrict network access (if not accepting external connections)
PrivateNetwork=false  # Already false by default

# Limit syscalls (advanced)
SystemCallFilter=@system-service

# Make /tmp private
PrivateTmp=true
```

## Troubleshooting

### Service Won't Start

```bash
# Check service status
sudo systemctl status lmproxy

# View recent logs
sudo journalctl -u lmproxy -n 50 --no-pager

# Verify config syntax
/opt/lmproxy/lmproxy /opt/lmproxy/config.yaml  # Run manually to see errors
```

### Common Issues

**Issue:** "Permission denied" when starting
- Fix: `sudo chown -R lmproxy:lmproxy /opt/lmproxy`

**Issue:** Config file not found
- Fix: Verify `/opt/lmproxy/config.yaml` exists
- Check with: `sudo test -f /opt/lmproxy/config.yaml && echo "OK" || echo "MISSING"`

**Issue:** Port already in use
- Fix: Change port in `config.yaml` or stop conflicting service

**Issue:** High memory usage
- Fix: Check `MemoryMax` limit, adjust as needed or investigate memory leak

### Logs Location

Logs go to systemd journal by default:
```bash
# All logs since boot
sudo journalctl -u lmproxy

# Real-time follow
sudo journalctl -u lmproxy -f

# Since specific time
sudo journalctl -u lmproxy --since "1 hour ago"

# Export to file
sudo journalctl -u lmproxy > lmproxy-logs.txt
```

## Auto-Restart Behavior

The service is configured to:
- Automatically restart on crash (`Restart=always`)
- Wait 5 seconds between attempts (`RestartSec=5`)
- Max restart rate limited by systemd

To disable auto-restart:
```ini
Restart=no
```

## Updating the Proxy

```bash
# 1. Stop service
sudo systemctl stop lmproxy

# 2. Build new version
go build -o lmproxy main.go

# 3. Backup old binary
sudo cp /opt/lmproxy/lmproxy /opt/lmproxy/lmproxy.backup

# 4. Install new binary
sudo cp lmproxy /opt/lmproxy/
sudo chown lmproxy:lmproxy /opt/lmproxy/lmproxy

# 5. Start service
sudo systemctl start lmproxy

# 6. Verify
sudo systemctl status lmproxy

# 7. Keep backup for 7 days, then remove
```

## Uninstall

```bash
# Stop and disable service
sudo systemctl stop lmproxy
sudo systemctl disable lmproxy

# Remove service file
sudo rm /etc/systemd/system/lmproxy.service

# Reload systemd
sudo systemctl daemon-reload

# Remove installation directory
sudo rm -rf /opt/lmproxy

# Remove service user (optional)
sudo deluser lmproxy
sudo delgroup lmproxy
```

## Integration with Firewall

If using `ufw`:
```bash
sudo ufw allow 9090/tcp  # Default port
sudo ufw reload
```

If using `firewalld`:
```bash
sudo firewall-cmd --permanent --add-port=9090/tcp
sudo firewall-cmd --reload
```
