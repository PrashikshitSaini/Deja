# Contributing to Deja

Thanks for helping improve Deja. Bug reports and focused pull requests are
welcome.

## Local checks

Deja requires Go 1.24 or newer and Zsh 5.8 or newer. Keep generated state in
the checkout while testing:

```zsh
mkdir -p .cache/go-build .cache/go-mod .tmp

GOCACHE="$PWD/.cache/go-build" \
GOMODCACHE="$PWD/.cache/go-mod" \
TMPDIR="$PWD/.tmp" \
go test -race ./...

GOCACHE="$PWD/.cache/go-build" \
GOMODCACHE="$PWD/.cache/go-mod" \
TMPDIR="$PWD/.tmp" \
go vet ./...

zsh -n shell/deja.zsh
sh -n scripts/install.sh scripts/package.sh
```

Build the current platform binary:

```zsh
GOCACHE="$PWD/.cache/go-build" \
GOMODCACHE="$PWD/.cache/go-mod" \
TMPDIR="$PWD/.tmp" \
go build -trimpath -o ./bin/deja ./cmd/deja
```

After building, check the shell transports and security regressions:

```zsh
zsh -dfi -c 'source shell/deja_test.zsh'
zsh -dfi -c 'source shell/async_test.zsh'
python3 shell/pty_test.py
```

The design and benchmark instructions for the session query worker are in
[docs/QUERY_WORKER.md](docs/QUERY_WORKER.md).

## Behavioral requirements

- Deja must never bind Enter or invoke Zsh's `accept-line` widget.
- Tab may only place a selected command into the editable buffer.
- Empty-prompt Up and Down must retain native Zsh history behavior.
- Commands beginning with a space must remain excluded from live recording.
- Tests, caches, binaries, and temporary stores must remain untracked.

Please include tests for behavior changes and run all checks before opening a
pull request.
