#!/bin/bash
set -e
# mpm installer

echo "mpm: Minecraft Plugin Manager Installer"

# Detect OS
os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m | tr '[:upper:]' '[:lower:]')
case $arch in
  x86_64) arch="amd64" ;;
  aarch64|arm64) arch="arm64" ;;
esac
# macOS uses Linux binaries
if [[ "$os" == "darwin" ]]; then
  os="linux"
fi

# Create directory
mkdir -p ~/.local/bin

# Download
echo "Downloading mpm..."
if ! curl -L "https://github.com/storrealbac/mpm/releases/latest/download/mpm-${os}-latest-${arch}.tar.gz" | tar -xz -C ~/.local/bin; then
  echo "Failed to download. Please check your internet connection."
  exit 1
fi

# Add to PATH
if [[ ":$PATH:" != *":$HOME/.local/bin:"* ]]; then
  echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
  if [ -f ~/.zshrc ]; then
    echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc
  fi
fi

echo "mpm installed to ~/.local/bin"
echo "Run 'mpm init' to get started"
echo "Restart your terminal to use mpm"