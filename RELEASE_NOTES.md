# Deja v0.3.2 - safer history and secret handling

Deja v0.3.2 adds deterministic secret redaction, safe command removal, and
hardening for terminal display and per-session palette state.

## Highlights

- Redacts sensitive environment assignments such as API keys, tokens,
  passwords, private keys, and credentials before display or insertion.
- Adds `deja purge`, a dry-run-first command for removing exact or literal
  matches from Deja's store and an optional Zsh history file.
- Escapes terminal control and bidirectional formatting characters for display
  while preserving the original command for insertion.
- Uses a unique, ownership-verified, mode-0700 runtime directory for every
  interactive shell session.
- Preserves shell operators, quoting, redirections, expansions, and spacing
  around redacted assignments.
- Prevents purge invocations from returning through live recording or history
  imports.

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/PrashikshitSaini/Deja/main/scripts/get.sh | sh
```

The installer detects macOS or Linux and ARM64 or AMD64, verifies the archive
checksum, installs Deja under the user's XDG data path, and prints the Zsh
configuration to add. It does not edit `.zshrc` itself.

## Assets

- `deja-v0.3.2-darwin-arm64.tar.gz` - Apple Silicon macOS
- `deja-v0.3.2-darwin-amd64.tar.gz` - Intel macOS
- `deja-v0.3.2-linux-arm64.tar.gz` - ARM64 Linux
- `deja-v0.3.2-linux-amd64.tar.gz` - AMD64 Linux
- `checksums-v0.3.2.txt` - SHA-256 checksums

See the [README](https://github.com/PrashikshitSaini/Deja#readme) for setup,
configuration, privacy details, troubleshooting, and source builds.
