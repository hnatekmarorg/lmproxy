#!/bin/bash
# Deploy LLM Proxy systemd service
set -e

SERVICE_NAME="lmproxy"
INSTALL_DIR="/opt/lmproxy"
BINARY_NAME="lmproxy"
CONFIG_FILE="config.yaml"

echo "=== LLM Proxy Deployment Script ==="

# Step 1: Build the binary
echo "[1/5] Building binary..."
go build -o "$BINARY_NAME" main.go

# Step 2: Create service user and installation directory
echo "[2/5] Creating service user and directories..."
sudo groupadd --system lmproxy 2>/dev/null || true
sudo useradd --system --no-create-home --gid lmproxy lmproxy 2>/dev/null || true
sudo mkdir -p "$INSTALL_DIR"
sudo mkdir -p "$INSTALL_DIR/logs"

# Step 3: Copy files
echo "[3/5] Copying files..."
sudo cp "$BINARY_NAME" "$INSTALL_DIR/"
if [ -f "$CONFIG_FILE" ]; then
    sudo cp "$CONFIG_FILE" "$INSTALL_DIR/"
else
    echo "Warning: $CONFIG_FILE not found in current directory"
fi

# Step 4: Install systemd service
echo "[4/5] Installing systemd service..."
sudo cp "${SERVICE_NAME}.service" /etc/systemd/system/

# Set proper permissions
sudo chmod 644 /etc/systemd/system/${SERVICE_NAME}.service
sudo chmod 755 "$INSTALL_DIR/$BINARY_NAME"
sudo chown -R lmproxy:lmproxy "$INSTALL_DIR"

# Step 5: Reload systemd and start service
echo "[5/5] Reloading systemd and starting service..."
sudo systemctl daemon-reload
sudo systemctl enable "${SERVICE_NAME}.service"

echo ""
echo "=== Deployment Complete ==="
echo ""
echo "Service installed at: /etc/systemd/system/${SERVICE_NAME}.service"
echo "Binary located at: $INSTALL_DIR/$BINARY_NAME"
echo "Config file at: $INSTALL_DIR/$CONFIG_FILE"
echo ""
echo "Management commands:"
echo "  Start:     sudo systemctl start ${SERVICE_NAME}"
echo "  Stop:      sudo systemctl stop ${SERVICE_NAME}"
echo "  Restart:   sudo systemctl restart ${SERVICE_NAME}"
echo "  Status:    sudo systemctl status ${SERVICE_NAME}"
echo "  Logs:      journalctl -u ${SERVICE_NAME} -f"
echo ""
echo "To start the service now, run: sudo systemctl start ${SERVICE_NAME}"
