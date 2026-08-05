# Commands

| Group     | Subcommands                                         |
| --------- | ---------------------------------------------------- |
| `db`      | create, drop, list, status, truncate                |
| `table`   | create, drop, list, truncate                        |
| `migrate` | run, status, make, rollback, reset, refresh, fresh  |
| `seed`    | run, make, list                                     |
| `app`     | build, serve, test, clean, install, status, version |

Every group has its own `help` subcommand (`gorun db help`, `gorun migrate
help`, ...), and `gorun help` / `gorun -c` / `gorun -i` cover the rest.

## `db`

Operates on whole databases on the resolved engine's server (see
[configuration.md](configuration.md) for engine/database resolution).

- `db create` / `db drop` - create or drop `DBConnConfig.DatabaseName`.
  `drop` shows the resolved target and asks for confirmation unless
  `--force`.
- `db list` - lists every database on the server.
- `db status` - connection + database existence check.
- `db truncate` - truncates every table in the resolved database; same
  confirmation behavior as `drop`.

## `table`

Operates on tables inside the resolved database.

- `table create` / `table drop` - create or drop a specific table.
- `table list` - lists tables in the resolved database.
- `table truncate` - empties a table without dropping it.

## `migrate`

SQL migrations, `goose`-style up/down files under
`MigrationPath/<engine>`.

- `migrate make <name>` - scaffolds a new timestamped migration file.
- `migrate run` - applies pending migrations.
- `migrate rollback` - reverts the last batch.
- `migrate reset` - reverts every migration.
- `migrate refresh` - reset then run.
- `migrate fresh` - drops all tables then run (not just a rollback - a
  clean rebuild).
- `migrate status` - shows applied vs. pending migrations.

`migrate reset`/`rollback`/`fresh` are destructive and go through the same
confirmation as `db drop`/`table drop`. `fresh --seed` and `refresh --seed`
also run seeders afterward - see [seeders.md](seeders.md) for what that
requires (`RunnerPath`, when running from the global CLI).

## `seed`

Runs `Seeder` implementations discovered under `SeederPath/<engine>` -
see [seeders.md](seeders.md) for the full interface and how registration
works.

- `seed run` - runs all seeders in registry order. Refused when `AppEnv`
  is `prod`/`production` unless `--force`.
- `seed make <name>` - scaffolds a new seeder file.
- `seed list` - lists discovered seeders and their resolved run order.

## `app`

Application lifecycle commands, independent of any database config.

- `app build` / `app serve` / `app test` / `app clean` / `app install` -
  thin wrappers around the equivalent `go build`/`go run`/`go test`/`go
  clean`/`go install` for `Config.ServerEntrypoint` (defaults to
  `cmd/server/main.go` - set it if your layout differs).
- `app status` / `app version` - environment/version info, no build step.
