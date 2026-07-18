# Releasing Deja

This document is for project maintainers. End-user installation and usage live
in the root README.

## Build release artifacts

Update the version in `cmd/deja/main.go`, `CHANGELOG.md`, and versioned user
documentation. Then run the checks in `CONTRIBUTING.md` and package all four
supported targets:

```zsh
./scripts/package.sh
```

The packager creates macOS and Linux archives for ARM64 and AMD64, plus a
SHA-256 checksum manifest, under `dist/`. To build selected targets:

```zsh
DEJA_TARGETS="darwin/arm64 linux/amd64" ./scripts/package.sh
```

Each archive must contain the binary, Zsh plugin, default configuration,
installer, README, changelog, and MIT license.

## Validate artifacts

Verify checksums from inside `dist/`:

```zsh
cd dist
shasum -a 256 -c checksums-v<VERSION>.txt
```

Extract an archive matching the current machine into `.tmp/`, run
`bin/deja version`, run `doctor` with temporary configuration and storage paths,
and smoke-test the packaged installer without writing outside the checkout.

## macOS signing and notarization

The packager signs macOS binaries with the hardened runtime when
`DEJA_CODESIGN_IDENTITY` names an installed Developer ID Application identity:

```zsh
DEJA_CODESIGN_IDENTITY="Developer ID Application: Name (TEAMID)" \
  ./scripts/package.sh
```

A public macOS release should also be submitted to Apple's notary service and
verified before publication. Notarization credentials must remain outside the
repository. Never commit certificates, private keys, passwords, API keys, or
temporary keychains.

## Publish

1. Commit the release changes and push `main`.
2. Create and push an annotated `v<VERSION>` tag.
3. Create a GitHub Release using the matching release notes.
4. Upload all four archives and the checksum manifest.
5. Verify that the release is public, non-draft, non-prerelease, and that every
   asset digest matches the locally generated artifact.

Keep the MIT `LICENSE` in every source and binary distribution.
