# Contributing

Internal process notes for working on `gorun` itself - not needed to
*use* the package (see [`docs/`](docs) and [README.md](README.md) for
that).

## Branches

| Branch | Purpose |
|---|---|
| `main` | Default branch. All work lands here first via PR. Always green: CI (`build`, `vet`, `test -race -cover`, `gofmt`, `golangci-lint`) must pass. |
| `release` | Production - what's actually tagged and published. Only updated by merging `main` into it. Pushing to `release` triggers the release automation below. Protect this branch (require PRs, no direct pushes) in the repo's GitHub settings. |
| `feature/<name>`, `fix/<name>` | Short-lived, branched off `main`, one change each, PR'd back into `main`. |
| `hotfix/<name>` | Branched off `release` for an urgent production-only fix; PR'd into `release` first, then merged/cherry-picked back into `main` so `main` doesn't regress. |

There's no long-lived `develop` branch - `main` already plays that role.
Keeping `release` as the only other durable branch matches this
package's own "no ceremony beyond what's needed" design (see `MultiDB`,
engine resolution, etc.).

## Cutting a release

1. Merge whatever should ship into `main` via normal PRs; CI runs on
   every one.
2. Open a PR from `main` into `release` and merge it.
3. That push to `release` runs `.github/workflows/release.yml`:
   - re-runs the full CI suite as a safety net,
   - auto-bumps and pushes a semver tag (default: patch),
   - creates a GitHub Release with auto-generated notes,
   - cross-compiles `gorun` for linux/darwin/windows (amd64/arm64) and
     attaches the binaries to the release.

No manual tagging, changelog-drafting-by-hand, or binary building - the
only manual step is the `main` -> `release` PR itself.

### Controlling the version bump

The tag bump defaults to **patch**. Include one of these tokens in a
commit message on the PR being merged to `release` to bump differently:

- `#major` - breaking change (`vX.0.0`)
- `#minor` - new backwards-compatible feature (`vX.Y.0`)
- `#patch` - bug fix / no functional change (default, explicit is fine too)

## Versioning

[Semantic Versioning](https://semver.org/): `vMAJOR.MINOR.PATCH`.

- **MAJOR** - breaking changes to the `Config` struct, `Seeder`/
  `SeederRegistry` interfaces, or CLI command/flag behavior.
- **MINOR** - new commands, new `Config` fields, new flags - additive,
  non-breaking.
- **PATCH** - bug fixes, doc updates, internal refactors with no
  observable behavior change.

While the module stays at `v0.x`, minor versions may still include
breaking changes (normal under SemVer for pre-1.0 software) - check
[CHANGELOG.md](CHANGELOG.md) before upgrading. Once this reaches
`v1.0.0`, a MAJOR bump beyond `v1` also requires updating the module path
to `.../v2` per Go modules' own major-version convention.

Every entry worth telling consumers about goes in
[CHANGELOG.md](CHANGELOG.md) under `[Unreleased]` as it lands on `main`,
then gets its version heading when that work is released.

## Linting

`golangci-lint run ./...` (config in [`.golangci.yml`](.golangci.yml))
runs in CI on every push/PR to `main`/`release`. Run it locally before
pushing - `gofmt`/`goimports` issues and most lint findings are cheap to
fix early, expensive to untangle later.
