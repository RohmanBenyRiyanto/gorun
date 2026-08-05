# gorun full sample

A complete walkthrough of the globally-installed `gorun` CLI - not the
library API (see [`../main.go`](../main.go) for that), the actual `gorun
setup` → migrate → seed workflow, built entirely with the real installed
binary against a local MySQL. Everything in this directory is the output
of the commands below, kept as a working reference.

This is its own Go module (own `go.mod`), `replace`-d to build against the
gorun code in this repo rather than a published version - a real consumer
project would just `go get github.com/RohmanBenyRiyanto/gorun` normally
and skip the `replace` line entirely.

## How this was built

```
go install github.com/RohmanBenyRiyanto/gorun/cmd/gorun@latest

mkdir full-sample && cd full-sample
go mod init github.com/you/full-sample

gorun setup --yes \
  --name gorun-sample \
  --mysql-host=127.0.0.1 --mysql-port=3306 --mysql-user=root \
  --mysql-database=gorun_sample
# writes .gorun/config.yaml, scaffolds database/migrations,
# database/seeders, and cmd/gorun-runner/main.go

gorun migrate make create_users_table
# edit the generated database/migrations/mysql/*.sql - see it in this repo

gorun migrate run --force
# real: creates the users table

# write database/seeders/mysql/users_seeder.go (see it in this repo),
# then register it in database/seeders/mysql/registry.go, right next to
# it - not in cmd/gorun-runner/main.go, so main.go never has to change
# as seeders are added

gorun seed run
# real: inserts the seeded rows via the runner cmd/gorun-runner delegates to
```

Every one of those ran against a real local MySQL - `gorun migrate run`
actually created `users`, `gorun seed run` actually inserted the three
rows below. Re-run it yourself:

```
cd example/full-sample
go run ./cmd/gorun-runner db status
go run ./cmd/gorun-runner migrate run --force
go run ./cmd/gorun-runner seed run
```

(`go run ./cmd/gorun-runner ...` here instead of a bare `gorun ...` only
because this directory isn't itself where you ran `go install` - a real
project just uses `gorun` directly once it's on `PATH`.)

## What's in here

| Path | What it is |
|---|---|
| `.gorun/config.yaml` | Written by `gorun setup` - MySQL connection, `runner_path` |
| `database/migrations/mysql/*.sql` | Written by `gorun migrate make`, filled in by hand |
| `database/seeders/mysql/users_seeder.go` | A real `Seeder` - three rows, no fixtures framework needed |
| `database/seeders/mysql/registry.go` | This package's `SeederRegistry` - registers `UsersSeeder`, lives next to it |
| `cmd/gorun-runner/main.go` | Scaffolded by `gorun setup`; just passes `mysqlseeders.Registry{}` to `Config.MySQLSeeders` |

## Result

```
mysql> SELECT * FROM users;
+----+----------------------+-------------------+-------+---------------------+
| id | name                 | email             | role  | created_at          |
+----+----------------------+-------------------+-------+---------------------+
|  1 | Rohman Beny Riyanto  | beny@example.com  | admin | 2026-08-05 01:26:02 |
|  2 | Ada Lovelace         | ada@example.com   | member| 2026-08-05 01:26:02 |
|  3 | Grace Hopper         | grace@example.com | member| 2026-08-05 01:26:02 |
+----+----------------------+-------------------+-------+---------------------+
```
