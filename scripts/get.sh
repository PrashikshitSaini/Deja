#!/bin/sh

# Deja remote installer.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/PrashikshitSaini/Deja/main/scripts/get.sh | sh
#
# Environment overrides:
#   DEJA_VERSION   Install a specific version, e.g. v0.3.1 (default: latest release)
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
CHECKSUMS_URL="https://github.com/$REPO/releases/download/${VERSION}/checksums-${VERSION}.txt"

TMPDIR_DEJA=$(mktemp -d)
trap 'rm -rf "$TMPDIR_DEJA"' EXIT HUP INT TERM
ARCHIVE="$TMPDIR_DEJA/${PACKAGE}.tar.gz"
CHECKSUMS="$TMPDIR_DEJA/checksums.txt"

echo "Downloading Deja ${VERSION} for ${os}/${arch}..."
if command -v curl >/dev/null 2>&1; then
  curl -fsSL "$URL" -o "$ARCHIVE"
  curl -fsSL "$CHECKSUMS_URL" -o "$CHECKSUMS"
elif command -v wget >/dev/null 2>&1; then
  wget -qO "$ARCHIVE" "$URL"
  wget -qO "$CHECKSUMS" "$CHECKSUMS_URL"
fi

expected=""
while read -r digest filename; do
  if [ "$filename" = "${PACKAGE}.tar.gz" ]; then
    expected=$digest
    break
  fi
done < "$CHECKSUMS"

if [ -z "$expected" ]; then
  echo "get.sh: release checksums do not contain ${PACKAGE}.tar.gz" >&2
  exit 1
fi

if command -v shasum >/dev/null 2>&1; then
  actual=$(shasum -a 256 "$ARCHIVE")
elif command -v sha256sum >/dev/null 2>&1; then
  actual=$(sha256sum "$ARCHIVE")
else
  echo "get.sh: need shasum or sha256sum to verify the release" >&2
  exit 1
fi
actual=${actual%% *}

if [ "$actual" != "$expected" ]; then
  echo "get.sh: checksum verification failed for ${PACKAGE}.tar.gz" >&2
  exit 1
fi

echo "Verified ${PACKAGE}.tar.gz"
tar -xzf "$ARCHIVE" -C "$TMPDIR_DEJA"

[ -d "$TMPDIR_DEJA/$PACKAGE" ] || {
  echo "get.sh: downloaded archive did not contain $PACKAGE" >&2
  exit 1
}

"$TMPDIR_DEJA/$PACKAGE/install.sh"
