# Deja v0.3.0 — find the command you meant

Deja turns Zsh history into an interactive command palette. Start typing a
command family, compare the variations you previously ran, scroll through old
and rare matches, and press Tab to bring one back into the prompt for editing.

Deja is local-only: there is no cloud service, account, telemetry, or remote
history processing. It never binds Enter and never executes a selection.

## Highlights

- Browse distinct command variants with Up and Down instead of seeing only one
  history match at a time.
- Scroll a compact six-row viewport across as many as 100 ranked candidates.
- Keep both ends of long commands visible with terminal-aware middle shortening.
- Rank matches using command family, text, directory, success, frequency, and
  recency.
- Use safe Tab insertion that leaves the command editable and unexecuted.
- Redact old Git commit messages before display or reinsertion.
- Configure candidate limits, age and frequency filters, hidden commands,
  allowlists, metadata, and rank styling.
- Import standard and `EXTENDED_HISTORY` Zsh history, then record new runs with
  working directory, status, and duration.

## Install

Download the archive matching your platform, verify it with
`checksums-v0.3.0.txt`, extract it, and run:

```zsh
./install.sh
```

Then add the lines printed by the installer to `~/.zshrc`, open a new terminal,
and run:

```zsh
deja doctor
```

## Rename note

The public name is **Deja** and the command is `deja`. Existing pre-release
`DEZA_*` environment variables remain accepted in v0.3.0 as migration
fallbacks, while `DEJA_*` is used for all new setup.

## Assets

- `deja-v0.3.0-darwin-arm64.tar.gz` — Apple Silicon macOS
- `deja-v0.3.0-darwin-amd64.tar.gz` — Intel macOS
- `deja-v0.3.0-linux-arm64.tar.gz` — ARM64 Linux
- `deja-v0.3.0-linux-amd64.tar.gz` — AMD64 Linux
- `checksums-v0.3.0.txt` — SHA-256 checksums

See the [README](https://github.com/PrashikshitSaini/Deja#readme) for the full
installation guide, configuration reference, privacy model, troubleshooting,
and packaging documentation.
