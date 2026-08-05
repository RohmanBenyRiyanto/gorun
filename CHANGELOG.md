# Changelog

All notable changes to this project are documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/) - see
[CONTRIBUTING.md](CONTRIBUTING.md) for how versions are decided and cut.

## [Unreleased]

### Fixed

- `gorun setup` scaffolded migration/seeder directories at the hardcoded
  `database/migrations`/`database/seeders` defaults regardless of
  `--mysql-migration-path`/`--mysql-seeder-path` (and the postgresql
  equivalents) actually being set to something else - a project with
  migrations/seeders living elsewhere got a second, empty set of
  directories next to its real ones. Now respects whatever path was
  actually configured.

### Added

- `gorun setup` now also scaffolds `gorun`/`gorun.bat` next to
  `cmd/gorun-runner` - a shortcut for `go run ./cmd/gorun-runner "$@"` so
  a database-touching command can be typed as `./gorun migrate run`
  instead of the full `go run` invocation every time. Same conditions as
  the runner it shortcuts to: only when a database engine is configured,
  never overwrites an existing file.
- `db`/`migrate`/`table` (previously just `seed`) now also delegate to
  `RunnerPath` when `.gorun/config.yaml` carries no connection info of
  its own for either engine - so a project that deliberately keeps that
  file minimal (just `name`/`runner_path`, to avoid duplicating
  connection settings it already gets from its own config system) still
  gets every command working from the bare global binary, no `./`-prefix
  workaround needed. When the file does carry connection info directly,
  that's still used as-is (faster, no `go run` subprocess) - this is
  purely a fallback for when it doesn't.

## [0.1.0] - 2026-08-05

Initial release.
