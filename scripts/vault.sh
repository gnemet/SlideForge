#!/bin/bash

# 🛡️ Digital Blacksmith: Vault Manager
# This script handles GPG symmetric encryption for sensitive configuration files.

ACTION=$1 # lock, unlock, or status
FILE=$2   # target file (e.g., .env)

usage() {
    echo "Usage: $0 [lock|unlock|status] [file_path]"
    echo "Example: $0 lock .env"
    echo ""
    echo "Note: It is recommended to set VAULT_PASS environment variable for non-interactive use."
    exit 1
}

if [[ -z "$ACTION" || -z "$FILE" ]]; then
    usage
fi

check_gpg() {
    if ! command -v gpg &> /dev/null; then
        echo "❌ Error: gpg is not installed."
        exit 1
    fi
}

lock() {
    check_gpg
    if [[ ! -f "$FILE" ]]; then
        echo "❌ Error: File '$FILE' not found."
        exit 1
    fi

    echo "🔒 Encrypting $FILE..."
    if [[ -n "$VAULT_PASS" ]]; then
        gpg --symmetric --cipher-algo AES256 --batch --yes --passphrase="$VAULT_PASS" "$FILE"
    else
        gpg --symmetric --cipher-algo AES256 "$FILE"
    fi

    if [[ -f "$FILE.gpg" ]]; then
        echo "✅ Created $FILE.gpg"
        echo "💡 You can now safely commit $FILE.gpg to the repository."
    else
        echo "❌ Encryption failed."
        exit 1
    fi
}

unlock() {
    check_gpg
    if [[ ! -f "$FILE.gpg" ]]; then
        echo "❌ Error: Encrypted file '$FILE.gpg' not found."
        exit 1
    fi

    echo "🔓 Decrypting $FILE.gpg..."
    if [[ -n "$VAULT_PASS" ]]; then
        gpg --decrypt --batch --yes --passphrase="$VAULT_PASS" "$FILE.gpg" > "$FILE"
    else
        gpg --decrypt "$FILE.gpg" > "$FILE"
    fi

    if [[ $? -eq 0 ]]; then
        echo "✅ Restored $FILE"
    else
        echo "❌ Decryption failed."
        exit 1
    fi
}

status() {
    if [[ -f "$FILE" && -f "$FILE.gpg" ]]; then
        echo "🔍 Status for '$FILE': Both raw and encrypted files exist."
        # Compare timestamps
        if [[ "$FILE" -nt "$FILE.gpg" ]]; then
            echo "⚠️  Warning: Raw file is newer than encrypted file. Run '$0 lock $FILE' to update."
        else
            echo "✨ Everything is up to date."
        fi
    elif [[ -f "$FILE.gpg" ]]; then
        echo "🔍 Status for '$FILE': Only encrypted file exists. Run '$0 unlock $FILE' to restore raw file."
    elif [[ -f "$FILE" ]]; then
        echo "🔍 Status for '$FILE': Only raw file exists. Run '$0 lock $FILE' to protect it."
    else
        echo "🔍 Status for '$FILE': No files found."
    fi
}

case "$ACTION" in
    lock) lock ;;
    unlock) unlock ;;
    status) status ;;
    *) usage ;;
esac
