#!/bin/sh
# Installs the latest BTCMenuBar into /Applications and opens it.
#   curl -fsSL https://raw.githubusercontent.com/jamalxcode/btconmenubar/main/install.sh | sh
set -eu
URL="https://github.com/jamalxcode/btconmenubar/releases/latest/download/BTCMenuBar.zip"
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT
echo "Downloading BTCMenuBar…"
curl -fsSL "$URL" -o "$TMP/BTCMenuBar.zip"
ditto -x -k "$TMP/BTCMenuBar.zip" "$TMP"
pkill -x btconmenubar 2>/dev/null || true
rm -rf /Applications/BTCMenuBar.app
ditto "$TMP/BTCMenuBar.app" /Applications/BTCMenuBar.app
xattr -dr com.apple.quarantine /Applications/BTCMenuBar.app 2>/dev/null || true
open /Applications/BTCMenuBar.app
echo "Installed. Look for ₿ in your menu bar."
