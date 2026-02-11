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
# Professional Mount Logic: Use external STORAGE_REMOTE if defined
MOUNT_DIR=${STORAGE_REMOTE:-"/home/gnemet/SlideForgeFiles/remote"}

# Antigravity Robust Mount Pattern: Detect broken FUSE endpoints early
if mountpoint -q "$MOUNT_DIR" 2>/dev/null; then
    if ! ls "$MOUNT_DIR" >/dev/null 2>&1; then
        echo "Detected broken mount at $MOUNT_DIR (Transport endpoint is not connected). Force unmounting..."
        fusermount -uz "$MOUNT_DIR" || umount -l "$MOUNT_DIR"
        sleep 1
    fi
fi

mkdir -p "$MOUNT_DIR"
if ! mountpoint -q "$MOUNT_DIR"; then
    echo "Mounting Google Drive Business/BDO to $MOUNT_DIR..."
    rclone mount google_drive:Business/BDO "$MOUNT_DIR" --daemon --vfs-cache-mode writes
    # Wait for mount to stabilize
    sleep 2
else
    echo "Google Drive already mounted and accessible at $MOUNT_DIR"
fi

# 3. Handle Cleanup of Obsolete Internal Folders
# If we have a legacy ./mnt folder that is NOT a mountpoint, remove it/warn
if [ -d "./mnt" ] && ! mountpoint -q "./mnt"; then
    echo "Cleaning up obsolete internal ./mnt folder..."
    rm -rf "./mnt"
fi
if [ -d "./storage" ]; then
    echo "Cleaning up obsolete internal ./storage folder..."
    rm -rf "./storage"
fi
if [ -d "./temp" ]; then
    echo "Cleaning up obsolete internal ./temp folder..."
    rm -rf "./temp"
fi

# 4. Building SlideForge...
go build -o bin/slideforge ./cmd/server

APP_PORT=${PORT:-8088}
LOG_FILE=${STORAGE_LOG:-"/home/gnemet/SlideForgeFiles/slideforge.log"}

echo "Killing process on port $APP_PORT..."
fuser -k $APP_PORT/tcp 2>/dev/null || true

echo "Starting SlideForge on http://localhost:$APP_PORT (Logs: $LOG_FILE)"
# We run in foreground but also tee to the log file
./bin/slideforge 2>&1 | tee -a "$LOG_FILE"
