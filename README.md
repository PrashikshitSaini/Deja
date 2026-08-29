# Deja

### Your shell remembers every command. Deja helps you find the right one.

[![Release](https://img.shields.io/github/v/release/PrashikshitSaini/Deja)](https://github.com/PrashikshitSaini/Deja/releases/latest)
[![CI](https://github.com/PrashikshitSaini/Deja/actions/workflows/ci.yml/badge.svg)](https://github.com/PrashikshitSaini/Deja/actions/workflows/ci.yml)
[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](./LICENSE)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](./CONTRIBUTING.md)

Deja turns Zsh history into a fast, interactive command palette. Type the start
of a command, see every variation you ran before side by side, scroll through
old and rare matches, and press Tab to bring one back for editing.

**No cloud. No telemetry. No daemons. No dependencies. No command execution.
Just your history, made useful.**

```text
$ git
Deja · 1-6 of 32 variants · ↑/↓ scroll · Tab insert
›  1  git [push]                                      ×24
   2  git [commit] [-m] [<message>]                   ×22
   3  git [commit] [-m] [<message>] [-m] [<message>]  ×10
   4  git [push] [origin] [main]                       ×6
   5  git [status]                                     ×6
   6  git [commit] [-am] [<message>]                   ×4
```

Unlike one-result-at-a-time history search, Deja shows you the possibilities
together so you can compare them. The six rows are a viewport, not a cutoff:
keep scrolling to browse as many as 100 ranked variants, including old one-use
commands.

---

## How Deja differs

Shell-history tools optimize for different workflows. `fzf` provides a fast,
general-purpose fuzzy finder; Atuin adds a rich history database and optional
encrypted sync; McFly provides context-aware full-screen history search; HSTR
focuses on searching and managing history; and zsh-autosuggestions offers one
unobtrusive ghost suggestion while you type.

Deja is specifically designed for comparing the ways you run a command without
leaving the prompt. It combines:

- **An inline palette while you type.** Start entering `git`, `docker`, `cd`, or
  another command family; no separate Ctrl-R interface is required.
- **Distinct variants with usage counts.** Repeated commands are grouped while
  meaningful flag and argument combinations remain visible side by side.
- **Token-level difference highlighting.** Brackets identify exactly which
  parts change between the visible variants.
- **Local, explainable ranking.** Text match, command family, frequency,
  current directory, success rate, and recency determine the order.
- **Insert-only selection.** Tab places a command in the editable prompt. Deja
  never binds Enter and never executes the selection.
- **A deliberately small local architecture.** One static Go binary, one Zsh
  integration, and a user-owned JSON Lines store. There is no runtime network
  behavior or telemetry.

Use Atuin when encrypted history sync across machines is the priority. Use fzf
when you want a programmable fuzzy finder for history, files, and other data.
Use zsh-autosuggestions when a single low-profile suggestion is enough. Use
Deja when you want to compare command variants directly in the prompt before
choosing one.

---

## Features

- Native Go binary with zero third-party runtime or build dependencies.
- Zsh integration that works inside regular terminal applications.
- Imports plain and `EXTENDED_HISTORY` Zsh history.
- Records new commands with working directory, exit status, and duration.
- Groups duplicate executions into distinct variants with usage counts.
- Ranks by text match, command family, current directory, success, frequency,
  and recency.
- Highlights tokens that differ between visible variants.
- Scrolls a viewport through a large ranked candidate pool (up to 100+ rows).
- Redacts sensitive environment assignments and reusable values such as old
  Git commit messages before display or insertion.
- Hides commands by family, prefix, or regular expression.
- Dim rank numbers, ANSI colors, row count, and metadata are configurable.
- User-private event store, lock, and result files.

## Safety contract

Deja deliberately does **not** bind Enter and never invokes Zsh's
`accept-line` widget. Selecting a result only changes `BUFFER`, the editable
command line. You can inspect or modify it before pressing Enter yourself.

When the prompt is empty and no Deja results are active, Up and Down fall back
to ordinary Zsh history navigation.

## Requirements

| | Supported |
| --- | --- |
| OS | macOS (Apple Silicon, Intel) · Linux (arm64, amd64) |
| Shell | Zsh 5.8+ |
| Windows | Not directly; works under WSL2 with Zsh |

Building from source additionally requires Go 1.24+.

## Install

Pick whichever route matches your setup. All routes install the same prebuilt,
statically linked binary.

### One-line install (macOS and Linux)

```sh
curl -fsSL https://raw.githubusercontent.com/PrashikshitSaini/Deja/main/scripts/get.sh | sh
```

The script detects your platform, downloads the latest release, installs Deja
under `${XDG_DATA_HOME:-$HOME/.local/share}/deja`, links `deja` into
`$HOME/.local/bin`, and prints the exact lines to add to your `~/.zshrc`:

```zsh
export PATH="$HOME/.local/bin:$PATH"
export DEJA_CONFIG="$HOME/.config/deja/config.json"
source "$HOME/.local/share/deja/shell/deja.zsh"
```

To pin a version instead of the latest release:

```sh
curl -fsSL https://raw.githubusercontent.com/PrashikshitSaini/Deja/main/scripts/get.sh | DEJA_VERSION=v0.3.2 sh
```

> **Note:** piping curl to sh runs third-party code with your user's
> privileges. Read [`scripts/get.sh`](./scripts/get.sh) first if that bothers
> you — it is short, and every step it takes is printed above.

### Manual install

Download the archive for your platform from
[GitHub Releases](https://github.com/PrashikshitSaini/Deja/releases/latest),
verify it against `checksums.txt`, then:

```sh
tar -xzf deja-v0.3.2-darwin-arm64.tar.gz
cd deja-v0.3.2-darwin-arm64
./install.sh
```

Available archives: `deja-v0.3.2-{darwin,linux}-{arm64,amd64}.tar.gz`.

### Build from source

```sh
git clone https://github.com/PrashikshitSaini/Deja.git
cd Deja
go build -trimpath -ldflags="-s -w" -o ./bin/deja ./cmd/deja
./scripts/install.sh   # after staging bin/, shell/, and deja.json together
```

Or try it straight from a checkout:

```sh
export DEJA_STORE="$PWD/.deja-dev/local-history.jsonl"
source "$PWD/shell/deja.zsh"
```

### Finish setup

Start a new Zsh session, then verify everything:

```sh
deja doctor
```

### macOS security notice

macOS release binaries are not yet signed or notarized. After verifying the
release checksum, you may need to approve the binary under **System Settings →
Privacy & Security → Open Anyway**, or build from source instead. Do not
disable Gatekeeper globally.

## Daily use

Type part of a command without pressing Enter:

```zsh
git
```

The palette updates as the editable buffer changes.

| Key | With Deja results | Without Deja results |
| --- | --- | --- |
| Up | Previous result; wraps and scrolls | Ordinary Zsh history |
| Down | Next result; wraps and scrolls | Ordinary Zsh history |
| Tab | Insert selection without executing | Ordinary Zsh completion |
| Enter | Normal execution — always yours | Normal execution |
| Continued typing | Refresh and rerank | Edit normally |
| Ctrl-C | Cancel the prompt | Cancel the prompt |

### Reading a result row

```text
 1  git [status] [--short]  ×8 · 100% ok · here
```

- `1` — rank position (dim; the top row is the best-ranked variant)
- `[brackets]` — tokens that differ between visible variants
- `×8` — this variant occurred eight times
- `100% ok` — recorded success rate where exit statuses exist
- `here` — has been run in the current directory

Deja escapes stored terminal control and bidirectional formatting characters
before display while preserving the original command for insertion. The live
palette uses a plain dim rank number; the CLI (`deja query --color always`)
renders only Deja's own ANSI colors.

## How ranking works

Six local, explainable signals, in priority order:

1. Exact command-family matches beat family-prefix matches.
2. Exact text > prefix text > contained text bonuses.
3. Bounded frequency bonus.
4. Current-directory bonus.
5. Small success-rate bonus.
6. Time-decaying recency bonus.

No remote service. No embedding model. No opaque score.

## Configuration

Configuration is a small JSON file, validated on every query — most edits apply
on the next keystroke. Resolution order: `$DEJA_CONFIG`, the `deja.json` beside
your installation, then `${XDG_CONFIG_HOME}/deja/config.json`.

```sh
deja config path     # which file is active
deja config check    # validate
deja config show     # dump effective settings
deja config init --file "$HOME/.config/deja/config.json"
```

Key settings (all optional):

| Setting | Default | Meaning |
| --- | ---: | --- |
| `display.limit` | `6` | Visible rows (1–50) |
| `display.candidate_limit` | `100` | Ranked variants kept for scrolling |
| `display.minimum_uses` | `1` | Hide rare variants |
| `display.minimum_query_length` | `1` | Keystrokes before the palette opens |
| `commands.only_families` | `[]` | Allowlist; when set, everything else hides |
| `commands.hidden_families` / `_prefixes` / `_patterns` | sensible defaults | Filter rules |
| `commands.redact_environment_variables` | common secret variable names | Redact matching `NAME=value` assignments |
| `commands.redact_flag_values` | git commit messages | Replace values before display |

Example — hide force pushes and anything carrying a password:

```json
{
  "commands": {
    "hidden_prefixes": ["git push --force"],
    "hidden_patterns": ["(?i)--password(?:=|\\s)"]
  }
}
```

### Sensitive environment assignments

Deja deterministically redacts assignment values when the variable name matches
one of `commands.redact_environment_variables`. The default case-insensitive
pattern covers names containing `API_KEY`, `ACCESS_KEY`, `TOKEN`, `SECRET`,
`PASSWORD`, `PASSWD`, `PRIVATE_KEY`, `CLIENT_SECRET`, or `CREDENTIAL`.

For example:

```text
export OPENAI_API_KEY=sk-private
```

is displayed as `export OPENAI_API_KEY=<redacted>`, and Tab inserts
`export OPENAI_API_KEY=""`. The original value remains in the raw history until
it is purged. Extend the regex list for organization-specific variable names;
set it to an empty list only when intentionally disabling this protection.
Assignments whose values contain dynamic shell syntax, such as command
substitution or arrays, are hidden entirely when they cannot be safely reduced
to one redacted token.

Deja does not guess based on value shape or entropy. A secret stored under an
unrelated variable name or passed as a plain positional argument requires a
custom redaction rule or purge.

See the full option reference further down in
[docs](./docs) or via `deja config show`.

## CLI reference

```sh
deja query --format plain --color always git status   # inspect ranked results
deja import --history-file "$HOME/.zsh_history"       # import existing history
deja purge --contains OPENAI_API_KEY                  # dry-run a store purge
deja stats                                            # store summary
deja doctor                                           # installation health check
deja config explain -- 'docker login --password x'    # preview redaction/filtering
deja version
```

Re-importing history is safe; event identities prevent duplicates.

### Purging commands

`purge` removes matching events from Deja's store and, when explicitly given,
a Zsh history file. It is a dry run unless `--force` is present:

```zsh
deja purge --contains OPENAI_API_KEY --history-file "$HISTFILE"
deja purge --contains OPENAI_API_KEY --history-file "$HISTFILE" --force
```

Use stdin when the exact command itself contains a secret, so the purge command
line does not repeat that secret:

```zsh
print -rn -- 'export OPENAI_API_KEY=sk-private' |
  deja purge --stdin --history-file "$HISTFILE" --force
```

`--exact` matches a complete command; `--contains` matches literal text; and
`--ignore-case` applies to either. Rewrites use atomic replacement and mode
`0600`, and Deja checks for concurrent changes before replacement. This cannot
coordinate with another shell writing at the same instant: close other active
Zsh sessions before purging so they cannot lose a new entry or write a removed
command back. Purging is irreversible; rotate a disclosed credential first.

## Data and privacy

- Zero network calls. Zero telemetry. This is structural, not a policy.
- History lives in a local JSON Lines file with user-only permissions.
- Commands starting with a space are ignored by the live hook.
- Display redaction and hidden commands do not modify raw history. Use
  `deja purge` when a stored command must be removed.
- Don't put secrets on command lines; prefer stdin, env files, or keychains —
  that advice predates and outlives any history tool.

## Uninstall

Remove the Deja lines from `~/.zshrc`, then:

```sh
rm -rf "${XDG_DATA_HOME:-$HOME/.local/share}/deja"
rm -f "$HOME/.local/bin/deja"
rm -rf "${XDG_CONFIG_HOME:-$HOME/.config}/deja"   # optional: removes config too
```

Review paths first if you customized `DEJA_CONFIG`, `DEJA_STORE`, or XDG vars.

## Troubleshooting

Run `deja doctor` first, then `deja config check`. Common issues:

- **Palette doesn't appear** — confirm interactive Zsh (`echo $ZSH_VERSION`),
  the binary runs (`deja version`), the store has events (`deja stats`), and
  your query meets `display.minimum_query_length`.
- **Keys don't respond** — re-source the plugin; load Deja *after* other
  keymap-touching plugins.
- **Too noisy** — raise `minimum_query_length` to 2–3, raise
  `minimum_uses`, or hide families/prefixes in config.

## Contributing

Bug reports and pull requests are welcome — see
[CONTRIBUTING.md](./CONTRIBUTING.md). Release engineering lives in
[docs/RELEASING.md](./docs/RELEASING.md). CI builds and tests on macOS and
Linux for every push.

## License

[MIT](./LICENSE) © 2026 Prashikshit Saini
