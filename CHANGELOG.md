# Changelog

All notable changes to this project are documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/) - see
[CONTRIBUTING.md](CONTRIBUTING.md) for how versions are decided and cut.

## [Unreleased]

### Added

- `gorun setup` now also scaffolds `gorun`/`gorun.bat` next to
  `cmd/gorun-runner` - a shortcut for `go run ./cmd/gorun-runner "$@"` so
  a database-touching command can be typed as `./gorun migrate run`
  instead of the full `go run` invocation every time. Same conditions as
  the runner it shortcuts to: only when a database engine is configured,
  never overwrites an existing file.

## [0.1.0] - 2026-08-05

Initial release.
