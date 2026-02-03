#!/bin/bash

# SlideForge Build and Run Script

# Mount Google Drive Business/BDO
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

# Link application directories to the mount (Antigravity Library Pattern)
# 'original' on GDrive -> 'uploads' locally
# 'template' on GDrive -> 'templates' locally
# 'thumbnails' on GDrive -> 'thumbnails' locally

manage_link() {
    local target=$1
    local link=$2
    if [ -L "$link" ]; then rm "$link"; fi
    if [ -d "$link" ]; then mv "$link" "$link.bak-$(date +%s)"; fi # Move existing real dirs
    ln -s "$target" "$link"
    echo "Linked $link -> $target"
}

manage_link "$MOUNT_DIR/original" "uploads"
manage_link "$MOUNT_DIR/template" "templates"
manage_link "$MOUNT_DIR/thumbnails" "thumbnails"

echo "Building SlideForge..."
go build -o bin/slideforge cmd/server/main.go

echo "Starting SlideForge on http://localhost:8088"
./bin/slideforge
