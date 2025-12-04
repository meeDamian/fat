#!/bin/bash
# Deploy script for FAT to Raspberry Pi with bidirectional sync
set -e

PI_HOST="10.10.10.10"
PI_USER="pi"
PI_PATH="/home/pi/fat"
LOCAL_PORT="4445"
REMOTE_PORT="4444"

echo "📥 Syncing database FROM Pi..."
rsync -avz --progress "$PI_USER@$PI_HOST:$PI_PATH/fat.db" ./fat.db 2>/dev/null || echo "  (no remote db yet)"

echo "📥 Syncing /h/ FROM Pi..."
rsync -avz --progress "$PI_USER@$PI_HOST:$PI_PATH/h/" ./h/ 2>/dev/null || echo "  (no remote h/ yet)"

echo ""
echo "🔨 Cross-compiling for ARM64 (Pi)..."
GOOS=linux GOARCH=arm64 go build -o fat-arm64 ./cmd/fat

echo "🔨 Building local binary..."
go build -o fat ./cmd/fat

echo ""
echo "📤 Uploading to $PI_USER@$PI_HOST..."
rsync -avz --progress fat-arm64 "$PI_USER@$PI_HOST:$PI_PATH/fat"
rm fat-arm64

echo "📤 Syncing database TO Pi..."
rsync -avz --progress ./fat.db "$PI_USER@$PI_HOST:$PI_PATH/"

echo "📤 Syncing /h/ TO Pi..."
rsync -avz --progress ./h/ "$PI_USER@$PI_HOST:$PI_PATH/h/" 2>/dev/null || true

echo ""
echo "🔄 Triggering remote reload..."
curl -s "http://$PI_HOST:$REMOTE_PORT/die" || true

echo "🔄 Triggering local reload (if running)..."
curl -s "http://localhost:$LOCAL_PORT/die" 2>/dev/null || true

echo ""
echo "✅ Deploy complete!"
echo "   Remote: http://$PI_HOST:$REMOTE_PORT"
echo "   Local:  http://localhost:$LOCAL_PORT (run: ./fat)"
