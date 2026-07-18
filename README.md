# Deja

### Your shell remembers every command. Deja helps you find the right one.

[![Release](https://img.shields.io/github/v/release/PrashikshitSaini/Deja)](https://github.com/PrashikshitSaini/Deja/releases)
[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](./LICENSE)

Deja turns Zsh history into a fast, interactive command palette. Type the start
of a command, see the useful variations you ran before, scroll through old and
rare matches, and press Tab to bring one back for editing.

No cloud. No telemetry. No command execution. Just your history, made useful.

```text
$ git
Deja · 1-6 of 32 variants · ↑/↓ scroll · Tab insert
› 🔴 git [push]                                      ×24
  🔻 git [commit] [-m] [<message>]                   ×22
  🔶 git [commit] [-m] [<message>] [-m] [<message>]  ×10
  ⭐ git [push] [origin] [main]                       ×6
  · git [status]                                     ×6
  · git [commit] [-am] [<message>]                   ×4
```

Unlike one-result-at-a-time history search, Deja lets you compare the
possibilities. The six rows are a viewport, not a cutoff: keep pressing Up or
Down to browse as many as 100 ranked variants by default, including old
one-use commands.

## Why people use Deja

- **Recover the exact flags.** Find the variation that worked without rebuilding
  a long command from memory.
- **Compare before choosing.** See distinct commands together instead of cycling
  through one opaque match at a time.
- **Stay in flow.** Use the arrow keys to browse and Tab to insert directly into
  the prompt you are already editing.
- **Keep control.** Deja never binds Enter and never runs a selected command.
- **Keep history private.** Everything stays in a local, user-owned event store.

Deja is a native Go binary with a Zsh integration and no third-party runtime
dependencies. Current version: `0.3.0`.

## See it in action

Typing a family such as `git`, `docker`, `kubectl`, `ffmpeg`, `aws`, `cd`, or
`ls` opens the palette automatically:

Use Up and Down to scroll through results. Tab places the selection in the
editable prompt so you can inspect or change it. Enter remains entirely under
your control.

## Features

- Native Go binary with no runtime or third-party Go dependencies.
- Zsh integration that works in regular terminal applications.
- Imports plain and `EXTENDED_HISTORY` Zsh history.
- Records new commands with working directory, exit status, and duration.
- Groups duplicate executions into distinct variants and shows usage counts.
- Ranks by text match, command family, current directory, success, frequency,
  and recency.
- Highlights tokens that differ between visible variants.
- Scrolls a small viewport through a larger ranked candidate pool.
- Shortens long rows in the middle while preserving their beginning and end.
- Hides commands by family, prefix, or regular expression.
- Supports optional family allowlists.
- Redacts reusable values such as old Git commit messages.
- Provides configurable rank markers, ANSI CLI colors, and metadata.
- Makes no network requests and has no telemetry.
- Keeps the event store, lock, and result files user-private.

## Safety contract

Deja deliberately does not bind Enter and never invokes Zsh's `accept-line`
widget. Selecting a result only changes `BUFFER`, the editable command line.
You can inspect or modify it before pressing Enter yourself.

When the prompt is empty and no Deja results are active, Up and Down fall back
to ordinary Zsh history navigation.

## Requirements

For a prebuilt release:

- Zsh 5.8 or newer.
- macOS or Linux on ARM64 or AMD64.

For building from source:

- Go 1.24 or newer.
- Zsh 5.8 or newer.

Windows is not currently supported. Deja's shell integration is Zsh-specific,
and its local store locking currently targets Unix-like systems.

## Install Deja

Download the latest release from
[GitHub Releases](https://github.com/PrashikshitSaini/Deja/releases/latest).

### Install a packaged release

Download the archive matching the operating system and CPU:

```text
deja-v0.3.0-darwin-arm64.tar.gz  # Apple Silicon Mac
deja-v0.3.0-darwin-amd64.tar.gz  # Intel Mac
deja-v0.3.0-linux-arm64.tar.gz
deja-v0.3.0-linux-amd64.tar.gz
```

Verify the archive against `checksums-v0.3.0.txt`, then extract and install it:

```zsh
tar -xzf deja-v0.3.0-darwin-arm64.tar.gz
cd deja-v0.3.0-darwin-arm64
./install.sh
```

The installer:

- Copies Deja to `${XDG_DATA_HOME:-$HOME/.local/share}/deja`.
- Links the CLI at `$HOME/.local/bin/deja`.
- Creates `${XDG_CONFIG_HOME:-$HOME/.config}/deja/config.json` if it does not
  already exist.
- Preserves an existing user configuration during upgrades.
- Prints the exact Zsh configuration to add; it does not edit `.zshrc` itself.

Add the following to `~/.zshrc`:

```zsh
export PATH="$HOME/.local/bin:$PATH"
export DEJA_CONFIG="$HOME/.config/deja/config.json"
source "$HOME/.local/share/deja/shell/deja.zsh"
```

Start a new Zsh session or reload the file:

```zsh
source ~/.zshrc
deja doctor
```

### macOS security notice

The v0.3.0 macOS binaries are not yet Apple Developer ID signed or notarized.
After verifying the release checksum, macOS users who choose to continue may
need to approve Deja under **System Settings → Privacy & Security → Open
Anyway**. Apple explains the risk and override process in its
[support guide](https://support.apple.com/guide/mac-help/open-a-mac-app-from-an-unknown-developer-mh40616/mac).

Do not disable Gatekeeper globally. Building from source is also supported
below for users who prefer a locally compiled binary.

## Build and try the source checkout

The following commands keep build caches, temporary files, the binary, and the
test history store inside this checkout:

```zsh
cd /path/to/Deja
mkdir -p .cache/go-build .cache/go-mod .tmp .deja-dev

GOCACHE="$PWD/.cache/go-build" \
GOMODCACHE="$PWD/.cache/go-mod" \
TMPDIR="$PWD/.tmp" \
go build -trimpath -o ./bin/deja ./cmd/deja

export DEJA_STORE="$PWD/.deja-dev/local-history.jsonl"
source "$PWD/shell/deja.zsh"
```

The checkout's [`deja.json`](./deja.json) is selected automatically by the Zsh
plugin. Existing Zsh history is imported in the background when the plugin is
sourced.

To activate the checkout automatically in later sessions, add absolute paths
to `~/.zshrc`:

```zsh
export DEJA_STORE="/absolute/path/to/Deja/.deja-dev/local-history.jsonl"
source "/absolute/path/to/Deja/shell/deja.zsh"
```

## Daily use

Type part of a command without pressing Enter:

```zsh
git
```

The palette updates as the editable buffer changes.

| Key | With Deja results | Without Deja results |
| --- | --- | --- |
| Up | Select the previous result; wrap and scroll when needed | Ordinary Zsh history |
| Down | Select the next result; wrap and scroll when needed | Ordinary Zsh history |
| Tab | Insert the selected command without executing it | Ordinary Zsh completion |
| Enter | Normal Zsh execution | Normal Zsh execution |
| Continued typing | Refresh and rerank the palette | Edit normally |
| Ctrl-C | Cancel the current prompt | Cancel the current prompt |

The header shows which part of the result set is visible. For example,
`7-12 of 32 variants` means that the seventh through twelfth ranked candidates
are currently displayed.

### Understanding a result row

```text
🔴 git [status] [--short]  ×8 · 100% ok · here
```

- `🔴` is the top rank marker.
- Brackets identify tokens that differ between the ranked variants.
- `×8` means this exact or redaction-normalized variant occurred eight times.
- `100% ok` is the recorded success rate where exit statuses are available.
- `here` means the command has been run in the current directory.

The default live rank markers are:

| Rank | Marker | Configured color name |
| ---: | :---: | --- |
| 1 | 🔴 | `dark-red` |
| 2 | 🔻 | `red` |
| 3 | 🔶 | `orange` |
| 4 | ⭐ | `gold` |
| 5+ | · | `default` |

ZLE sanitizes raw terminal control sequences in its message area, so the live
palette uses safe geometric markers. Direct CLI output can color complete rows
with ANSI sequences by using `--color always`.

## How ranking works

Ranking uses local, explainable signals:

1. Exact command-family matches are preferred over family-prefix matches.
2. Exact text, prefix text, and contained text matches receive decreasing
   bonuses.
3. Frequently used variants receive a bounded frequency bonus.
4. Variants used in the current directory receive a directory bonus.
5. Successful variants receive a small success-rate bonus.
6. Recent variants receive a time-decaying recency bonus.

Ranking does not call a remote service and does not use an embedding model.

## Configuration

Deja uses JSON configuration version 1. Configuration is loaded for every
query, so most edits apply on the next keystroke without rebuilding or
restarting the shell.

The plugin uses this resolution order:

1. `DEJA_CONFIG`, when explicitly set.
2. The `deja.json` beside an extracted or source installation.
3. `${XDG_CONFIG_HOME}/deja/config.json`.
4. `$HOME/.config/deja/config.json`.

Validate and inspect the active configuration:

```zsh
deja config path
deja config check
deja config show
```

Create a default file at a chosen path:

```zsh
deja config init --file "$HOME/.config/deja/config.json"
```

Use `--force` only when intentionally replacing an existing file.

### Display settings

| Setting | Default | Meaning |
| --- | ---: | --- |
| `display.limit` | `6` | Visible rows in the scrolling viewport; valid range 1-50 |
| `display.candidate_limit` | `100` | Ranked variants kept for scrolling; must be at least `limit`, maximum 5000 |
| `display.minimum_uses` | `1` | Hide variants used fewer times than this value |
| `display.maximum_age_days` | `0` | Exclude timestamped runs older than this many days; zero disables the age limit |
| `display.minimum_query_length` | `1` | Do not open the palette until the trimmed query reaches this length |
| `display.metadata.usage` | `true` | Show the `×N` usage count |
| `display.metadata.success_rate` | `true` | Show success percentage when status data exists |
| `display.metadata.current_directory` | `true` | Show `here` for current-directory matches |
| `display.colors.enabled` | `true` | Enable live rank markers and configured ANSI output |
| `display.colors.rank` | six colors | Color name for each rank; the final entry repeats for later ranks |
| `display.colors.family` | `inherit` | CLI ANSI color for the command family token |
| `display.colors.difference` | `bright-yellow` | CLI ANSI color for differing tokens |
| `display.colors.metadata` | `dim` | CLI ANSI style for metadata |

Available color names are `inherit`, `default`, `black`, `dark-red`, `red`,
`bright-red`, `orange`, `gold`, `yellow`, `bright-yellow`, `green`, `cyan`,
`blue`, `magenta`, `white`, `gray`, `dim`, and `bold`.

`DEJA_LIMIT` can temporarily override `display.limit` for one shell:

```zsh
export DEJA_LIMIT=10
source /path/to/deja/shell/deja.zsh
```

### Command visibility settings

| Setting | Default | Meaning |
| --- | --- | --- |
| `commands.only_families` | `[]` | Optional allowlist; when nonempty, every other family is hidden |
| `commands.hidden_families` | common shell-management commands | Hide complete families such as `history` and `exit` |
| `commands.hidden_prefixes` | `[]` | Hide commands whose shell-token prefix matches a configured value |
| `commands.hidden_patterns` | `[]` | Hide commands matching a Go regular expression |
| `commands.redact_flag_values` | Git commit-message rule | Replace values associated with selected command flags |

Only show Git and Docker commands:

```json
"only_families": ["git", "docker"]
```

Hide force pushes and commands that expose a password argument:

```json
"hidden_prefixes": ["git push --force"],
"hidden_patterns": ["(?i)--password(?:=|\\s)"]
```

Patterns use Go regular-expression syntax. Because the configuration is JSON,
backslashes must be escaped as `\\`.

### Redacting flag values

The default configuration contains:

```json
{
  "command_prefix": "git commit",
  "flags": ["-m", "-am", "--message"],
  "display_placeholder": "<message>",
  "insert_placeholder": "\"\""
}
```

These commands:

```text
git commit -m "fix the checkout race"
git commit -m "add account validation"
```

are aggregated and shown as:

```text
git commit -m <message>  ×2
```

Tab inserts `git commit -m ""`, not an old message. The cursor remains in the
editable prompt, ready for a new message.

Test any rule without opening the palette:

```zsh
deja config explain -- \
  'git commit -m "a private message"'
```

Example output:

```text
visible: true
display: git commit -m <message>
insert: git commit -m ""
```

Redaction controls display, aggregation, and Tab insertion. It does not rewrite
the original Zsh history file or erase the raw event already stored by Deja.

## Environment variables

| Variable | Purpose |
| --- | --- |
| `DEJA_BIN` | Absolute binary override; otherwise the plugin checks its bundled binary and then `PATH` |
| `DEJA_CONFIG` | Absolute configuration-file override |
| `DEJA_STORE` | Absolute event-store override |
| `DEJA_LIMIT` | Temporary visible-row override |
| `XDG_CONFIG_HOME` | Base directory used for the default config path |
| `XDG_DATA_HOME` | Base directory used for the default event-store path |

### Migrating from the pre-release `Deza` name

The public product, command, and configuration names are now `Deja`, `deja`,
and `DEJA_*`. Version 0.3.0 still accepts the earlier `DEZA_BIN`, `DEZA_CONFIG`,
`DEZA_STORE`, and `DEZA_LIMIT` variables so an existing local history does not
disappear during the rename. When both names are present, `DEJA_*` wins.

For a permanent migration, update the variable names in `~/.zshrc`. To keep
using an existing event store, point the new variable at it:

```zsh
export DEJA_STORE="/absolute/path/to/your/existing/history.jsonl"
source "/absolute/path/to/Deja/shell/deja.zsh"
```

The default event store is:

```text
${XDG_DATA_HOME:-$HOME/.local/share}/deja/history.jsonl
```

## CLI reference

Run `deja` without arguments to see the top-level command list.

### `deja import`

Import plain or extended Zsh history. Re-importing the same entries is safe;
event identities prevent duplicates.

```zsh
deja import --history-file "$HOME/.zsh_history"
deja import --history-file "$HISTFILE" --store /custom/history.jsonl
```

### `deja query`

Inspect ranked variants without ZLE:

```zsh
deja query --format plain git status
deja query --format plain --color always git
deja query --format json --limit 20 docker
print -rn -- 'kubectl get' | deja query --query-stdin --format plain
```

Useful options include `--cwd`, `--limit`, `--visible-rows`, `--width`,
`--color`, `--store`, and `--config`. The `zle` format, `--zle-meta`,
`--results-file`, and `pick` command primarily support the shell plugin.

### `deja config`

```zsh
deja config path
deja config init --file /path/to/config.json
deja config check --file /path/to/config.json
deja config show --file /path/to/config.json
deja config explain --file /path/to/config.json -- 'docker login --password secret'
```

### `deja stats`, `doctor`, and `version`

```zsh
deja stats
deja doctor
deja version
```

`doctor` reports the binary version, Go runtime, Zsh path, store path, config
path, config status, and whether the tool is local-only.

### `deja record` and `pick`

These commands are public but primarily intended for integrations:

- `record` stores one completed command and optional runtime metadata.
- `pick` returns a cached candidate by one-based index without executing it.

## Data and privacy

- Deja makes no network calls and includes no telemetry.
- Imported and live commands are stored locally as JSON Lines.
- The event store and lock use user-only file permissions.
- Commands beginning with a space are ignored by the live Zsh hook.
- Whether a leading-space command also stays out of `.zsh_history` depends on
  the user's Zsh history options, such as `HIST_IGNORE_SPACE`.
- Display redaction does not remove data from the source history or raw store.
- Hiding a command prevents it from appearing in results; it does not delete an
  existing raw event.

Do not place secrets directly on command lines when a tool supports stdin,
environment files, keychains, or interactive secret entry instead.

## Troubleshooting

### Start with `doctor`

```zsh
deja doctor
```

Then validate the configuration:

```zsh
deja config check
```

### The palette does not appear

1. Confirm the shell is interactive Zsh with `echo $ZSH_VERSION`.
2. Confirm the binary is executable with `deja version`.
3. Confirm the store contains events with `deja stats`.
4. Type enough characters to satisfy `display.minimum_query_length`.
5. Check that allowlists, hidden families, prefixes, or patterns are not
   filtering the command.
6. Re-source `shell/deja.zsh` after changing installation paths.

### Up, Down, or Tab does not work

Re-source the plugin in the affected terminal:

```zsh
source /absolute/path/to/deja/shell/deja.zsh
```

Deja binds common application/cursor sequences and terminal-provided `terminfo`
sequences in the `main`, `emacs`, and `viins` keymaps. Another plugin sourced
later can replace those bindings, so load Deja after conflicting keybinding
plugins.

### Up and Down should use normal history

They do when the prompt is empty or Deja has no candidates. When candidates are
visible, the same keys scroll the Deja selection. Tab inserts a result and
closes the palette; it never submits it.

### The configuration is rejected

Deja rejects unknown JSON fields, invalid regular expressions, unsupported
colors, invalid ranges, and unsupported config versions. `deja config check`
prints the exact error. The plugin also validates configuration while loading
so an invalid file does not fail silently.

### The palette feels noisy or slow

- Raise `display.minimum_query_length` from `1` to `2` or `3`.
- Raise `display.minimum_uses` to hide one-off commands.
- Lower `display.candidate_limit` while keeping it at least as large as
  `display.limit`.
- Set `display.maximum_age_days` to exclude stale timestamped events.
- Hide noisy command families or prefixes.

### Rank markers instead of colored text

This is intentional in the live palette. ZLE's message area escapes raw ANSI
sequences, so Deja uses terminal-safe geometric rank markers. Use
`deja query --format plain --color always ...` to inspect ANSI text colors.

## Uninstall

Remove the Deja lines from `~/.zshrc`, start a new shell, and remove the local
installation:

```zsh
rm -rf "${XDG_DATA_HOME:-$HOME/.local/share}/deja"
rm -f "$HOME/.local/bin/deja"
```

To remove configuration and history as well:

```zsh
rm -rf "${XDG_CONFIG_HOME:-$HOME/.config}/deja"
rm -f "${XDG_DATA_HOME:-$HOME/.local/share}/deja/history.jsonl"
rm -f "${XDG_DATA_HOME:-$HOME/.local/share}/deja/history.jsonl.lock"
```

Review paths before deleting them, especially when `DEJA_CONFIG`, `DEJA_STORE`,
or the XDG variables are customized.

## Contributing

Bug reports and pull requests are welcome. See
[CONTRIBUTING.md](./CONTRIBUTING.md) for the local development checks. Release
engineering is documented separately in [docs/RELEASING.md](./docs/RELEASING.md).

## License

Deja is available under the [MIT License](./LICENSE).

```text
Copyright (c) 2026 Prashikshit Saini
```
