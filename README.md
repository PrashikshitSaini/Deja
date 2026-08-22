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
› 🔴 git [push]                                      ×24
  🔻 git [commit] [-m] [<message>]                   ×22
  🔶 git [commit] [-m] [<message>] [-m] [<message>]  ×10
  ⭐ git [push] [origin] [main]                       ×6
  · git [status]                                     ×6
  · git [commit] [-am] [<message>]                   ×4
```

Unlike one-result-at-a-time history search, Deja shows you the possibilities
together so you can compare them. The six rows are a viewport, not a cutoff:
keep scrolling to browse as many as 100 ranked variants, including old one-use
commands.

---

## Why Deja exists

Every shell history tool solves "find that command again." Most of them solve
it by adding weight to your shell:

| | Deja | fzf + history | Atuin | McFly | zsh-autosuggestions |
| --- | --- | --- | --- | --- | --- |
| Shows multiple variants at once | ✅ | ❌ one at a time | ❌ list of raw lines | ❌ one at a time | ❌ single ghost line |
| Groups duplicates into variants with counts | ✅ | ❌ | ❌ | ❌ | ❌ |
| Highlights what differs between variants | ✅ | ❌ | ❌ | ❌ | ❌ |
| Context-aware ranking (directory, success rate, recency) | ✅ | ❌ | ✅ | ✅ | ❌ |
| Never binds Enter / never executes | ✅ | ⚠️ easy to misfire | ⚠️ replaces history widget | ❌ rebinds Enter | ✅ |
| External daemon or background process | none | none | **daemon** (`atuin daemon`) | none | none |
| Runtime language | native Go binary | Rust binary | Rust binary + SQLite | Rust binary | pure Zsh |
| Database dependency | none (JSON Lines) | none | **SQLite** | SQLite | none |
| Syncs your history to the cloud | never | never | optional (encrypted) | never | never |
| Configuration surface | small JSON file | shell flags | TOML + sync accounts | config file | vars |

The pattern in the column is consistent: to get smarter history, existing tools
ask you to adopt a daemon, a database, a sync account, or a rewrite of how your
shell handles history. That is bloat if all you wanted was `git commit -am`
with the right flags.

Deja takes the opposite trade-off:

- **One static binary and one Zsh script.** No daemon, no SQLite, no plugins
  manager, no compile step on your machine.
- **Your history stays yours.** Everything is a local JSON Lines file with
  user-only permissions. There is no sync feature to turn off — there is no
  network code at all.
- **Enter is sacred.** Deja only ever edits the command line you are already
  typing. You review it; you press Enter. It cannot run anything.
- **Small, explainable ranking.** Six local signals — text match, family,
  frequency, current directory, success rate, recency. No embeddings, no
  model downloads, no mystery scores.

If you want encrypted multi-machine history search, Atuin is a good tool and
you should use it. If you want a fast general-purpose fuzzy finder, use fzf.
If you want your last matching command as a ghost suggestion with zero setup,
zsh-autosuggestions does that well. **If you want to see and compare the ways
you actually run a command — locally, instantly, and without adopting new
infrastructure — Deja is built for exactly that.**

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
- Redacts reusable values such as old Git commit messages before display.
- Hides commands by family, prefix, or regular expression.
- Configurable rank markers, ANSI colors, row count, and metadata.
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
curl -fsSL https://raw.githubusercontent.com/PrashikshitSaini/Deja/main/scripts/get.sh | DEJA_VERSION=v0.3.0 sh
```

> **Note:** piping curl to sh runs third-party code with your user's
> privileges. Read [`scripts/get.sh`](./scripts/get.sh) first if that bothers
> you — it is short, and every step it takes is printed above.

### Install with npm

```sh
npm install -g deja
```

The postinstall step downloads the correct prebuilt binary for your platform
from GitHub Releases and wires up the paths. Then add to `~/.zshrc`:

```zsh
export PATH="$(npm root -g)/deja/bin:$PATH"
export DEJA_CONFIG="$(npm root -g)/deja/deja.json"
source "$(npm root -g)/deja/shell/deja.zsh"
```

### Manual install

Download the archive for your platform from
[GitHub Releases](https://github.com/PrashikshitSaini/Deja/releases/latest),
verify it against `checksums.txt`, then:

```sh
tar -xzf deja-v0.3.0-darwin-arm64.tar.gz
cd deja-v0.3.0-darwin-arm64
./install.sh
```

Available archives: `deja-v0.3.0-{darwin,linux}-{arm64,amd64}.tar.gz`.

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
🔴 git [status] [--short]  ×8 · 100% ok · here
```

- `🔴` — rank marker (top-ranked variant)
- `[brackets]` — tokens that differ between visible variants
- `×8` — this variant occurred eight times
- `100% ok` — recorded success rate where exit statuses exist
- `here` — has been run in the current directory

ZLE sanitizes raw ANSI sequences, so the live palette uses geometric markers;
the CLI (`deja query --color always`) renders full ANSI colors.

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
| `display.candidate_pool` | `100` | Ranked variants kept for scrolling |
| `display.minimum_uses` | `1` | Hide rare variants |
| `display.minimum_query_length` | `1` | Keystrokes before the palette opens |
| `commands.only_families` | `[]` | Allowlist; when set, everything else hides |
| `commands.hidden_families` / `_prefixes` / `_patterns` | sensible defaults | Filter rules |
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

See the full option reference further down in
[docs](./docs) or via `deja config show`.

## CLI reference

```sh
deja query --format plain --color always git status   # inspect ranked results
deja import --history-file "$HOME/.zsh_history"       # import existing history
deja stats                                            # store summary
deja doctor                                           # installation health check
deja config explain -- 'docker login --password x'    # preview redaction/filtering
deja version
```

Re-importing history is safe; event identities prevent duplicates.

## Data and privacy

- Zero network calls. Zero telemetry. This is structural, not a policy.
- History lives in a local JSON Lines file with user-only permissions.
- Commands starting with a space are ignored by the live hook.
- Display redaction and hidden commands affect what you see, not what your
  shell's own history file records.
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
