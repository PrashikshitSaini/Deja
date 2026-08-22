#!/bin/sh

# Deja remote installer.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/PrashikshitSaini/Deja/main/scripts/get.sh | sh
#
# Environment overrides:
#   DEJA_VERSION   Install a specific version, e.g. v0.3.0 (default: latest release)
#   DEJA_INSTALL_DIR  Installation root (default: ${XDG_DATA_HOME:-$HOME/.local/share}/deja)
#   DEJA_BIN_DIR      Where the `deja` symlink is placed (default: $HOME/.local/bin)

set -eu

REPO="PrashikshitSaini/Deja"
VERSION=${DEJA_VERSION:-latest}

OS=$(uname -s)
ARCH=$(uname -m)

case "$OS" in
  Darwin) os=darwin ;;
  Linux) os=linux ;;
  *)
    echo "get.sh: unsupported operating system '$OS'" >&2
    echo "Deja supports macOS and Linux. On Windows use WSL2 with Zsh." >&2
    exit 1
    ;;
esac

case "$ARCH" in
  arm64|aarch64) arch=arm64 ;;
  x86_64|amd64) arch=amd64 ;;
  *)
    echo "get.sh: unsupported architecture '$ARCH' (need arm64 or amd64)" >&2
    exit 1
    ;;
esac

if [ "$VERSION" = "latest" ]; then
  if command -v curl >/dev/null 2>&1; then
    VERSION=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
      | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p')
  elif command -v wget >/dev/null 2>&1; then
    VERSION=$(wget -qO- "https://api.github.com/repos/$REPO/releases/latest" \
      | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p')
  else
    echo "get.sh: need curl or wget to download Deja" >&2
    exit 1
  fi
fi

if [ -z "$VERSION" ]; then
  echo "get.sh: could not determine the latest release version" >&2
  exit 1
fi

PACKAGE="deja-${VERSION}-${os}-${arch}"
URL="https://github.com/$REPO/releases/download/${VERSION}/${PACKAGE}.tar.gz"

TMPDIR_DEJA=$(mktemp -d)
trap 'rm -rf "$TMPDIR_DEJA"' EXIT HUP INT TERM

echo "Downloading Deja ${VERSION} for ${os}/${arch}..."
if command -v curl >/dev/null 2>&1; then
  curl -fsSL "$URL" | tar -xz -C "$TMPDIR_DEJA"
elif command -v wget >/dev/null 2>&1; then
  wget -qO- "$URL" | tar -xz -C "$TMPDIR_DEJA"
fi

[ -d "$TMPDIR_DEJA/$PACKAGE" ] || {
  echo "get.sh: downloaded archive did not contain $PACKAGE" >&2
  exit 1
}

"$TMPDIR_DEJA/$PACKAGE/install.sh"
