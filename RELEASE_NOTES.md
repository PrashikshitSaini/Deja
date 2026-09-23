# Deja v0.3.3 - smoother terminal typing

Deja v0.3.3 moves history searches off the shell's redraw path so typing no
longer waits for suggestions, even with a large history store.

## Highlights

- Asynchronous palette searches through one background worker per shell session.
- A 50 ms debounce combines rapid typing into fewer searches.
- An in-memory family index reuses parsed history and reloads when the store changes.
- Request IDs keep stale responses from replacing current suggestions or insertion text.
- Regression tests cover cache invalidation, stale responses, private runtime
  state, worker cleanup, and real-terminal display and Tab insertion.

The worker starts with the first nonempty query and stops when the shell exits.
There are no new dependencies or history-storage migrations. Cold searches and
searches after history changes still reload the store, but do so asynchronously.

## Contributor thanks

Thanks to [@0xkaushik-ai](https://github.com/0xkaushik-ai) for spotting the
performance problem in [#1](https://github.com/PrashikshitSaini/Deja/issues/1)
and following through with the async worker, session index, and tests in
[#2](https://github.com/PrashikshitSaini/Deja/pull/2). This release's performance
fix is their contribution.

## Updating

Run the installer below, then open a new terminal tab to load the updated shell
integration. Existing history and configuration are preserved.

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/PrashikshitSaini/Deja/main/scripts/get.sh | sh
```

The installer detects macOS or Linux and ARM64 or AMD64, verifies the archive
checksum, installs Deja under the user's XDG data path, and prints the Zsh
configuration to add. It does not edit `.zshrc` itself.

## Assets

- `deja-v0.3.3-darwin-arm64.tar.gz` - Apple Silicon macOS
- `deja-v0.3.3-darwin-amd64.tar.gz` - Intel macOS
- `deja-v0.3.3-linux-arm64.tar.gz` - ARM64 Linux
- `deja-v0.3.3-linux-amd64.tar.gz` - AMD64 Linux
- `checksums-v0.3.3.txt` - SHA-256 checksums

See the [README](https://github.com/PrashikshitSaini/Deja#readme) for setup,
configuration, privacy details, troubleshooting, and source builds.
