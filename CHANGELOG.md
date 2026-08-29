# Changelog

All notable changes to Deja are documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project uses [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.3.2] - 2026-08-29

### Added

- Automatic, configurable redaction of sensitive environment assignments such
  as API keys, access tokens, passwords, and client secrets. Display uses a
  `<redacted>` marker while Tab insertion uses an empty value.
- A dry-run-first `deja purge` command for atomically deleting exact or
  literal-matching commands from the Deja store and an optional Zsh history
  file.

### Safety

- Purge invocations are excluded from live recording and history imports so
  sensitive match text is not reintroduced after deletion.
- Zsh history purge preserves symlink paths and aborts if the target changes
  while its atomic replacement is prepared.
- Palette display escapes terminal control and bidirectional formatting
  characters without changing the command returned for insertion.
- Each interactive shell uses an atomically created, ownership-verified,
  mode-0700 runtime directory for palette result files.

## [0.3.1] - 2026-08-23

### Added

- One-line installer for macOS and Linux with automatic OS and architecture
  detection.
- GitHub Actions checks for macOS and Linux, plus tag-driven release packaging.

### Changed

- Replaced prominent emoji rank markers with compact numeric rank indicators.
- Split the CLI entry point into focused command files without changing the
  public command interface.
- Reworked the README around installation, product comparisons, safety, and
  privacy.

### Fixed

- Removed a forced ZLE redisplay after Tab insertion that could overwrite the
  terminal line immediately above the prompt.
- Quoted the Zsh integration path in tests so checkouts under paths containing
  spaces work correctly.

### Security

- The remote installer now verifies every downloaded archive against the
  release's SHA-256 checksum manifest before extraction.
- Palette result files now live in a per-user mode-0700 runtime directory
  instead of directly under a shared temporary directory.
- Removed an unpublished npm installation route that could resolve to an
  unrelated package owned by somebody else.

## [0.3.0] - 2026-07-18

### Added

- Native Go CLI for importing, recording, querying, and inspecting Zsh history.
- Interactive Zsh palette with Up/Down navigation and edit-first Tab insertion.
- A six-row scrolling viewport over as many as 100 ranked command variants.
- Distinct-variant grouping, frequency counts, recency and directory ranking,
  success-rate metadata, and token-difference highlighting.
- Middle truncation for long commands so both the beginning and ending remain
  visible in narrow terminals.
- JSON configuration for visibility rules, candidate limits, rank styling,
  metadata, age limits, minimum use counts, and command redaction.
- Safe Git commit-message redaction for display, aggregation, and insertion.
- Local-only JSONL storage with file locking and user-private permissions.
- `stats`, `doctor`, `version`, and configuration inspection commands.
- Cross-platform release archives for macOS and Linux on ARM64 and AMD64.
- Packaged installer, SHA-256 checksum manifest, MIT license, and full guide.

### Changed

- Renamed the pre-release product, command, and public configuration namespace
  from `Deza`/`deza`/`DEZA_*` to `Deja`/`deja`/`DEJA_*`.
- Kept the earlier `DEZA_*` environment variables as compatibility fallbacks for
  existing local users; the new names take precedence.

### Safety

- Deja never binds Enter, never invokes Zsh's `accept-line`, and never executes
  a selected result. Tab only replaces the editable command buffer.
- Empty-prompt Up/Down behavior falls back to native Zsh history navigation.
- Commands beginning with a space are excluded from live recording.

[Unreleased]: https://github.com/PrashikshitSaini/Deja/compare/v0.3.2...HEAD
[0.3.2]: https://github.com/PrashikshitSaini/Deja/compare/v0.3.1...v0.3.2
[0.3.1]: https://github.com/PrashikshitSaini/Deja/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/PrashikshitSaini/Deja/releases/tag/v0.3.0
