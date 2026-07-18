#!/bin/sh

set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$ROOT"
DIST=${DEJA_DIST_DIR:-${DEZA_DIST_DIR:-"$ROOT/dist"}}
TARGETS=${DEJA_TARGETS:-${DEZA_TARGETS:-"darwin/arm64 darwin/amd64 linux/arm64 linux/amd64"}}
VERSION=${DEJA_VERSION:-${DEZA_VERSION:-$(sed -n 's/^const version = "\([^"]*\)"/\1/p' "$ROOT/cmd/deja/main.go")}}

if [ -z "$VERSION" ]; then
  echo "package: could not determine the Deja version" >&2
  exit 1
fi

command -v go >/dev/null 2>&1 || {
  echo "package: Go is required" >&2
  exit 1
}
command -v tar >/dev/null 2>&1 || {
  echo "package: tar is required" >&2
  exit 1
}

mkdir -p "$ROOT/.cache/go-build" "$ROOT/.cache/go-mod" "$ROOT/.tmp" "$DIST"
STAGE=$(mktemp -d "$ROOT/.tmp/deja-package.XXXXXX")
trap 'rm -rf "$STAGE"' EXIT HUP INT TERM

for target in $TARGETS; do
  os=${target%/*}
  arch=${target#*/}
  package_name="deja-v${VERSION}-${os}-${arch}"
  package_dir="$STAGE/$package_name"
  archive="$DIST/$package_name.tar.gz"

  mkdir -p "$package_dir/bin" "$package_dir/shell"
  env \
    CGO_ENABLED=0 \
    GOOS="$os" \
    GOARCH="$arch" \
    GOCACHE="$ROOT/.cache/go-build" \
    GOMODCACHE="$ROOT/.cache/go-mod" \
    TMPDIR="$ROOT/.tmp" \
    go build -trimpath -ldflags="-s -w" -o "$package_dir/bin/deja" ./cmd/deja

  codesign_identity=${DEJA_CODESIGN_IDENTITY:-${DEZA_CODESIGN_IDENTITY:-}}
  if [ "$os" = "darwin" ] && [ -n "$codesign_identity" ]; then
    command -v codesign >/dev/null 2>&1 || {
      echo "package: codesign is required when DEJA_CODESIGN_IDENTITY is set" >&2
      exit 1
    }
    codesign --force --options runtime --timestamp \
      --sign "$codesign_identity" "$package_dir/bin/deja"
  fi

  cp "$ROOT/shell/deja.zsh" "$package_dir/shell/deja.zsh"
  cp "$ROOT/deja.json" "$package_dir/deja.json"
  cp "$ROOT/README.md" "$package_dir/README.md"
  cp "$ROOT/CHANGELOG.md" "$package_dir/CHANGELOG.md"
  cp "$ROOT/scripts/install.sh" "$package_dir/install.sh"
  if [ -f "$ROOT/LICENSE" ]; then
    cp "$ROOT/LICENSE" "$package_dir/LICENSE"
  fi
  chmod 0755 "$package_dir/bin/deja" "$package_dir/install.sh"
  tar -C "$STAGE" -czf "$archive" "$package_name"
  echo "created $archive"
done

checksums="$DIST/checksums-v${VERSION}.txt"
if command -v shasum >/dev/null 2>&1; then
  (
    cd "$DIST"
    shasum -a 256 deja-v"$VERSION"-*.tar.gz
  ) >"$checksums"
elif command -v sha256sum >/dev/null 2>&1; then
  (
    cd "$DIST"
    sha256sum deja-v"$VERSION"-*.tar.gz
  ) >"$checksums"
else
  echo "package: shasum or sha256sum is required for checksums" >&2
  exit 1
fi

echo "created $checksums"
