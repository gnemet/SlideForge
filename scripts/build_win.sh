#!/bin/bash

# SlideForge Windows Build Script (Premium Portability Edition)
# This script cross-compiles SlideForge for Windows 10/11 with embedded assets.

set -e

APP_NAME="SlideForge"
VERSION=$(grep -oP 'version: "\K[^"]+' config.yaml || echo "0.2.0")
DIST_DIR="dist_win"

echo "🔨 Building $APP_NAME v$VERSION for Windows..."

# 1. Create distribution directory
mkdir -p "$DIST_DIR"
mkdir -p "$DIST_DIR/uploads"
mkdir -p "$DIST_DIR/thumbnails"

# 2. Cross-compile Go binary
echo "🚀 Compiling Go binary (GOOS=windows)..."
GOOS=windows GOARCH=amd64 go build -o "$DIST_DIR/$APP_NAME.exe" ./cmd/server

# 3. Copy essential files
echo "📄 Copying configuration files..."
cp config.yaml "$DIST_DIR/config.yaml.template"

# 4. Create a RUN.bat for easy startup
echo "⚡ Creating startup script..."
cat <<EOF > "$DIST_DIR/RUN.bat"
@echo off
TITLE SlideForge Server
echo Starting SlideForge...
if not exist config.yaml (
    echo [!] No config.yaml found. Copying from template...
    copy config.yaml.template config.yaml
)
SlideForge.exe
pause
EOF

# 5. Summary
echo "------------------------------------------------"
echo "✅ Build Complete!"
echo "📍 Location: $DIST_DIR/"
echo "💼 To deploy, ZIP the '$DIST_DIR' folder and send it to the Windows target."
echo "------------------------------------------------"
echo "Requirements on Windows 11:"
echo "1. PostgreSQL 17 (installed or portable)"
echo "2. LibreOffice (installed, in PATH)"
echo "3. Poppler (pdftoppm, in PATH)"
echo "4. Ollama (for local LLM / Offline mode)"
