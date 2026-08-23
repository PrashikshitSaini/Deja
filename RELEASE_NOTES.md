# Deja v0.3.1 - safer installation and cleaner terminal behavior

Deja v0.3.1 is a focused reliability and release-safety update for the local
Zsh command-variant palette.

## Highlights

- Fixes a ZLE redraw issue where inserting a selection with Tab could repaint
  the terminal line immediately above the prompt.
- Verifies release archives against the published SHA-256 checksum manifest
  before the one-line installer extracts them.
- Stores transient palette results in a private per-user runtime directory.
- Replaces emoji rank markers with compact numeric ranks.
- Adds macOS and Linux CI plus automated release packaging.
- Removes the unavailable npm installation route so every documented install
  method resolves to Deja itself.

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/PrashikshitSaini/Deja/main/scripts/get.sh | sh
```

The installer detects macOS or Linux and ARM64 or AMD64, downloads the matching
archive, verifies its checksum, installs Deja under the user's XDG data path,
and prints the Zsh configuration to add. It does not edit `.zshrc` itself.

## Assets

- `deja-v0.3.1-darwin-arm64.tar.gz` - Apple Silicon macOS
- `deja-v0.3.1-darwin-amd64.tar.gz` - Intel macOS
- `deja-v0.3.1-linux-arm64.tar.gz` - ARM64 Linux
- `deja-v0.3.1-linux-amd64.tar.gz` - AMD64 Linux
- `checksums-v0.3.1.txt` - SHA-256 checksums

See the [README](https://github.com/PrashikshitSaini/Deja#readme) for setup,
configuration, privacy details, troubleshooting, and source builds.
