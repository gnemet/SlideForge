#!/bin/bash

# SlideForge Build and Run Script

# 1. Environment Selection logic (Antigravity Environment Pattern)
# Instead of hardcoding keys in .env, we select an environment file from opt/envs/
# and copy it to .env at runtime.

# Default environment
ENV_NAME="zenbook"

# Check if a specific environment arg was passed
if [ -n "$1" ]; then
    ENV_NAME="$1"
fi

ENV_SOURCE="opt/envs/.env_${ENV_NAME}"

if [ -f "$ENV_SOURCE" ]; then
    echo "Using environment configuration: $ENV_SOURCE"
    cp "$ENV_SOURCE" .env
else
    echo "Warning: Environment file $ENV_SOURCE not found!"
    # Fallback: check if .env already exists, otherwise warn
    if [ ! -f ".env" ]; then
        echo "No .env file found either. Application might fail if dependent on env vars."
    fi
fi

# Load variables for script usage (like port, storage paths)
if [ -f ".env" ]; then
    set -a
    source .env
    set +a
fi

# 2. Mount Google Drive Business/BDO
MOUNT_DIR="./mnt/bdo"
mkdir -p "$MOUNT_DIR"
if ! mountpoint -q "$MOUNT_DIR"; then
    echo "Mounting Google Drive Business/BDO to $MOUNT_DIR..."
    rclone mount google_drive:Business/BDO "$MOUNT_DIR" --daemon --vfs-cache-mode writes
    # Wait for mount to stabilize
    sleep 2
else
    echo "Google Drive already mounted at $MOUNT_DIR"
fi

# 3. Link application directories to the mount (Antigravity Library Pattern)
# Use env vars from the loaded .env file, or fallback defaults if something is missing.

STAGE_PATH=${STORAGE_STAGE:-"./mnt/bdo/stage"}
TEMPLATE_PATH=${STORAGE_TEMPLATE:-"./mnt/bdo/template"}
THUMB_PATH=${STORAGE_THUMBNAILS:-"./mnt/bdo/thumbnails"}

manage_link() {
    local target=$1
    local link=$2
    
    # Create target directory if it doesn't exist (important for fresh mounts)
    mkdir -p "$target"

    if [ -L "$link" ]; then rm "$link"; fi
    if [ -d "$link" ] && [ ! -L "$link" ]; then 
        echo "Backing up existing directory $link..."
        mv "$link" "$link.bak-$(date +%s)"
    fi
    
    ln -s "$target" "$link"
    echo "Linked $link -> $target"
}

manage_link "$STAGE_PATH" "uploads"
manage_link "$TEMPLATE_PATH" "templates"
manage_link "$THUMB_PATH" "thumbnails"

echo "Building SlideForge..."
go build -o bin/slideforge ./cmd/server

APP_PORT=${PORT:-8088}
echo "Starting SlideForge on http://localhost:$APP_PORT"
./bin/slideforge
