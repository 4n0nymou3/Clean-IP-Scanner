#!/data/data/com.termux/files/usr/bin/bash

set -e

clear
echo "=========================================="
echo "   Clean IP Scanner - Installer"
echo "=========================================="
echo ""
echo "Installing for Termux (Android ARM64)"
echo ""

echo "[1/6] Checking and installing packages..."
if ! command -v git &> /dev/null; then
    echo "  → Installing git..."
    pkg install -y git || { echo "✗ Failed to install git"; exit 1; }
fi
if ! command -v go &> /dev/null; then
    echo "  → Installing golang..."
    pkg install -y golang || { echo "✗ Failed to install golang"; exit 1; }
fi
if ! command -v curl &> /dev/null; then
    echo "  → Installing curl..."
    pkg install -y curl
fi
if ! command -v unzip &> /dev/null; then
    echo "  → Installing unzip..."
    pkg install -y unzip
fi
if ! command -v jq &> /dev/null; then
    echo "  → Installing jq..."
    pkg install -y jq
fi
echo "✓ All packages ready"

echo ""
echo "[2/6] Downloading source code..."
cd ~
if [ -d "Clean-IP-Scanner" ]; then
    echo "  → Removing old installation..."
    rm -rf Clean-IP-Scanner
fi
git clone -q https://github.com/4n0nymou3/Clean-IP-Scanner.git || { echo "✗ Failed to clone repository"; exit 1; }
cd Clean-IP-Scanner || { echo "✗ Directory not found"; exit 1; }
echo "✓ Source code downloaded"

echo ""
echo "[3/6] Downloading dependencies..."
go mod tidy || { echo "✗ Failed to download dependencies"; exit 1; }
echo "✓ Dependencies ready"

echo ""
echo "[4/6] Installing Xray core (latest stable for Android ARM64-v8a)..."

if [ -f "./xray/xray" ]; then
    echo "  → Xray binary already present, skipping download."
else
    MAX_RETRIES=3
    RETRY_COUNT=0

    download_xray() {
        echo "  → Fetching latest Xray version from GitHub API (attempt $((RETRY_COUNT + 1))/$MAX_RETRIES)..."

        LATEST_URL=$(curl -sL "https://api.github.com/repos/XTLS/Xray-core/releases/latest" | jq -r '.assets[] | select(.name=="Xray-android-arm64-v8a.zip") | .browser_download_url')

        if [ -z "$LATEST_URL" ] || [ "$LATEST_URL" = "null" ]; then
            echo "  → Could not get download URL from API"
            return 1
        fi

        echo "  → Downloading from $LATEST_URL"
        curl -L --retry 3 --retry-delay 5 -o xray-core.zip "$LATEST_URL" || { echo "  → Download failed"; return 1; }

        if [ ! -f xray-core.zip ] || [ ! -s xray-core.zip ]; then
            echo "  → Downloaded file is missing or empty"
            return 1
        fi

        unzip -o xray-core.zip -d xray_temp || { echo "  → Unzip failed"; rm -f xray-core.zip; return 1; }
        mkdir -p xray
        cp xray_temp/xray xray/
        chmod +x xray/xray
        rm -rf xray_temp xray-core.zip
        return 0
    }

    while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
        if download_xray; then
            echo "✓ Xray core installed"
            break
        else
            RETRY_COUNT=$((RETRY_COUNT + 1))
            if [ $RETRY_COUNT -lt $MAX_RETRIES ]; then
                echo "  → Retrying in 15 seconds..."
                sleep 15
            fi
        fi
    done

    if [ ! -f "./xray/xray" ]; then
        echo ""
        echo "  → Auto-detection failed. Trying fallback version..."
        FALLBACK_VERSION="v26.3.27"
        FALLBACK_URL="https://github.com/XTLS/Xray-core/releases/download/${FALLBACK_VERSION}/Xray-android-arm64-v8a.zip"
        echo "  → Downloading $FALLBACK_VERSION from $FALLBACK_URL"

        if curl -L --retry 3 -o xray-core.zip "$FALLBACK_URL" && \
           unzip -o xray-core.zip -d xray_temp && \
           mkdir -p xray && \
           cp xray_temp/xray xray/ && \
           chmod +x xray/xray; then
            rm -rf xray_temp xray-core.zip
            echo "✓ Xray core installed (fallback version $FALLBACK_VERSION)"
        else
            rm -rf xray_temp xray-core.zip
            echo "✗ Failed to install Xray core. Please check your internet connection."
            exit 1
        fi
    fi
fi

echo ""
echo "[5/6] Setting up Xray config files..."
mkdir -p config

if [ ! -f "config/xray_config.json" ]; then
    cat > config/xray_config.json << 'EOF'
{
  "log": { "loglevel": "warning" },
  "inbounds": [
    {
      "port": 1080,
      "protocol": "socks",
      "settings": { "udp": false },
      "listen": "127.0.0.1"
    }
  ],
  "outbounds": [
    {
      "protocol": "vless",
      "settings": {
        "vnext": [
          {
            "address": "IP_PLACEHOLDER",
            "port": 443,
            "users": [
              { "id": "your-uuid-here", "encryption": "none", "flow": "xtls-rprx-vision" }
            ]
          }
        ]
      },
      "streamSettings": {
        "network": "tcp",
        "security": "tls",
        "tlsSettings": {
          "serverName": "your-domain.com",
          "allowInsecure": false
        }
      }
    }
  ]
}
EOF
    echo "✓ Sample JSON config created at config/xray_config.json"
else
    echo "✓ Existing xray_config.json found, keeping it."
fi

if [ ! -f "config/xray_config.txt" ]; then
    cat > config/xray_config.txt << 'EOF'
# Xray URL Config
# Put your proxy URL on the line below (remove the # at the start).
# Supported formats: vless://, vmess://, trojan://, ss://
# Example:
# vless://your-uuid@your-server.com:443?type=ws&security=tls&host=your-server.com&path=%2F&sni=your-server.com#MyConfig
#
# If this file has a valid URL, it will be used instead of xray_config.json.
EOF
    echo "✓ Sample URL config created at config/xray_config.txt"
else
    echo "✓ Existing xray_config.txt found, keeping it."
fi

echo ""
echo "[6/6] Building clean-ip-scanner..."
echo "  (This may take 1-2 minutes...)"
CGO_ENABLED=0 go build -buildmode=pie -ldflags="-s -w" -o clean-ip-scanner || { echo "✗ Build failed"; exit 1; }
if [ ! -f "clean-ip-scanner" ]; then
    echo "✗ Build failed - executable not created"
    exit 1
fi
echo "✓ Build completed"

echo ""
echo "Installing to system..."
cat > $PREFIX/bin/clean-ip-scanner << 'SCRIPT'
#!/data/data/com.termux/files/usr/bin/bash
cd ~/Clean-IP-Scanner
./clean-ip-scanner "$@"
SCRIPT
chmod +x $PREFIX/bin/clean-ip-scanner
echo "✓ Installed to PATH"

echo ""
echo "=========================================="
echo "   Installation completed successfully!"
echo "=========================================="
echo ""
echo "Usage:"
echo "  clean-ip-scanner"
echo ""
echo "  You will be asked to choose:"
echo "    1) Scan mode:"
echo "         1 - Normal scan (TCP ping + speed test)"
echo "         2 - Xray scan (uses Xray core with your config)"
echo "    2) Worker count (Xray mode only)"
echo "    3) Scan port (press Enter for default port 443,"
echo "         or enter any port number between 1 and 65535)"
echo ""
echo "  For Xray mode, edit ONE of these files:"
echo "    URL format : ~/Clean-IP-Scanner/config/xray_config.txt"
echo "    JSON format: ~/Clean-IP-Scanner/config/xray_config.json"
echo ""
echo "  Results saved to: clean_ips.txt and clean_ips_list.txt"
echo ""
echo "You can now run: clean-ip-scanner"
echo ""