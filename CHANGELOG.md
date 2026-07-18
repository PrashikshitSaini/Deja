# Changelog

All notable changes to Deja are documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project uses [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/PrashikshitSaini/Deja/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/PrashikshitSaini/Deja/releases/tag/v0.3.0
