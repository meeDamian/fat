#!/bin/bash
# Deploy script for FAT to Raspberry Pi
set -e

PI_HOST="10.10.10.10"
PI_USER="pi"
PI_PATH="/home/pi/fat"

echo "🔨 Cross-compiling for ARM64..."
GOOS=linux GOARCH=arm64 go build -o fat ./cmd/fat

echo "📦 Uploading to $PI_USER@$PI_HOST..."
rsync -avz --progress fat "$PI_USER@$PI_HOST:$PI_PATH/"

echo "🔄 Triggering reload..."
curl -s "http://$PI_HOST:4444/die" || true

echo "✅ Deploy complete!"
