# gorun

A Laravel-artisan-style command-line toolkit for Go projects: database and
table management, SQL migrations, seeders, and a handful of application
lifecycle commands (build/serve/test/clean/install) - all wired up behind
one [urfave/cli/v3](https://github.com/urfave/cli) command tree.

Supports MySQL and PostgreSQL, either one alone or both side by side in
the same project. Use it as a library behind your own `main.go`, or
install it once as a global CLI that discovers a per-project config file
the same way `git` finds `.git/`.

## Install

As a library, to build your own CLI binary around it:

```
go get github.com/RohmanBenyRiyanto/gorun
```

Or as a ready-made global CLI:

```
go install github.com/RohmanBenyRiyanto/gorun/cmd/gorun@latest
```

## Quick start

```go
cfg := gorun.Config{
    MySQL: gorun.DBConnConfig{
        Host:          "127.0.0.1",
        Port:          "3306",
        User:          "root",
        Password:      os.Getenv("DB_PASSWORD"),
        DatabaseName:  "myapp",
        MigrationPath: "database/migrations",
        SeederPath:    "database/seeders",
    },
    MySQLSeeders: myapp.SeederRegistry{}, // your project's seeders
}

cmd := gorun.New(cfg)
if err := cmd.Run(context.Background(), os.Args); err != nil {
    log.Fatal(err)
}
```

`cmd` is a regular `*cli.Command`, so it composes normally if you want to
fold it into a bigger CLI of your own. Or skip writing config by hand and
let the global CLI scaffold everything:

```
gorun setup
```

## Support

| | |
|---|---|
| **Databases** | MySQL, PostgreSQL - either one alone, or both together with `MultiDB: true` (see [docs/configuration.md](docs/configuration.md)) |
| **OS** | Linux, macOS, Windows - `app build`/`app serve` shell out via `bash -c` on Linux/macOS and `cmd /C` on Windows |
| **Go** | 1.22 or newer (see [go.mod](go.mod)) |
| **CLI framework** | [urfave/cli/v3](https://github.com/urfave/cli) |
| **ORM/driver** | [gorm.io/gorm](https://gorm.io) over `gorm.io/driver/mysql` and `gorm.io/driver/postgres` |

## Commands

| Group     | Subcommands                                         |
| --------- | ---------------------------------------------------- |
| `db`      | create, drop, list, status, truncate                |
| `table`   | create, drop, list, truncate                        |
| `migrate` | run, status, make, rollback, reset, refresh, fresh  |
| `seed`    | run, make, list                                     |
| `app`     | build, serve, test, clean, install, status, version |

## Documentation

Full reference material lives in [`docs/`](docs):

- [docs/setup.md](docs/setup.md) - installing, `gorun setup`, config
  discovery
- [docs/configuration.md](docs/configuration.md) - every `Config` field,
  the YAML format, single- vs. multi-engine resolution
- [docs/commands.md](docs/commands.md) - every command group and
  subcommand in detail
- [docs/seeders.md](docs/seeders.md) - the `Seeder`/`SeederRegistry`
  interfaces and `RunnerPath` delegation

[CHANGELOG.md](CHANGELOG.md) has notable changes per release. See
[CONTRIBUTING.md](CONTRIBUTING.md) for the branch model, release process,
and versioning policy if you're working on `gorun` itself.

## Examples

Runnable, end-to-end setups under [`example/`](example):

- [`full-sample`](example/full-sample) - `gorun setup` -> migrate -> seed,
  done for real against a local MySQL, using the global binary.
- [`existing-config-sample`](example/existing-config-sample) - integrating
  `gorun` into a project that already has its own config system.
- [`single-db`](example/single-db) / [`multi-db`](example/multi-db) -
  minimal single-engine and dual-engine setups.

## License

See [LICENSE](LICENSE).
