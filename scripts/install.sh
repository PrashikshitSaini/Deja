#!/bin/sh

set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
if [ -x "$SCRIPT_DIR/bin/deja" ]; then
  PACKAGE_ROOT=$SCRIPT_DIR
else
  PACKAGE_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)
fi
DATA_HOME=${XDG_DATA_HOME:-"$HOME/.local/share"}
CONFIG_HOME=${XDG_CONFIG_HOME:-"$HOME/.config"}
INSTALL_DIR=${DEJA_INSTALL_DIR:-${DEZA_INSTALL_DIR:-"$DATA_HOME/deja"}}
BIN_DIR=${DEJA_BIN_DIR:-${DEZA_BIN_DIR:-"$HOME/.local/bin"}}
CONFIG_FILE=${DEJA_CONFIG:-${DEZA_CONFIG:-"$CONFIG_HOME/deja/config.json"}}

if [ ! -x "$PACKAGE_ROOT/bin/deja" ]; then
  echo "install: $PACKAGE_ROOT/bin/deja is missing or not executable" >&2
  exit 1
fi

mkdir -p "$INSTALL_DIR/bin" "$INSTALL_DIR/shell" "$BIN_DIR" "$(dirname -- "$CONFIG_FILE")"
cp "$PACKAGE_ROOT/bin/deja" "$INSTALL_DIR/bin/deja"
cp "$PACKAGE_ROOT/shell/deja.zsh" "$INSTALL_DIR/shell/deja.zsh"
cp "$PACKAGE_ROOT/README.md" "$INSTALL_DIR/README.md"
if [ -f "$PACKAGE_ROOT/CHANGELOG.md" ]; then
  cp "$PACKAGE_ROOT/CHANGELOG.md" "$INSTALL_DIR/CHANGELOG.md"
fi
cp "$PACKAGE_ROOT/deja.json" "$INSTALL_DIR/deja.json"
if [ -f "$PACKAGE_ROOT/LICENSE" ]; then
  cp "$PACKAGE_ROOT/LICENSE" "$INSTALL_DIR/LICENSE"
fi
chmod 0755 "$INSTALL_DIR/bin/deja"
ln -sfn "$INSTALL_DIR/bin/deja" "$BIN_DIR/deja"

if [ ! -e "$CONFIG_FILE" ]; then
  cp "$PACKAGE_ROOT/deja.json" "$CONFIG_FILE"
  chmod 0600 "$CONFIG_FILE"
  echo "created $CONFIG_FILE"
else
  echo "kept existing $CONFIG_FILE"
fi

echo "installed Deja in $INSTALL_DIR"
echo "linked the CLI at $BIN_DIR/deja"
echo ""
echo "Add these lines to ~/.zshrc:"
echo "export DEJA_CONFIG=\"$CONFIG_FILE\""
echo "source \"$INSTALL_DIR/shell/deja.zsh\""
echo ""
echo "Make sure $BIN_DIR is in PATH, then open a new Zsh session."
