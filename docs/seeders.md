# Seeders

gorun ships no seeders of its own, only the machinery to discover, order,
and run them. Implement `SeederRegistry` for your project's seeders and
wire it in through `Config.MySQLSeeders` / `Config.PostgreSQLSeeders`:

```go
type Seeder interface {
    Name() string
    Run(db *gorm.DB, sqlDB *sql.DB) error
}

type SeederRegistry interface {
    RegisterSeeder(typeName string) (reflect.Type, bool)
    GetSeederOrder() []string
}
```

`GetSeederOrder` is the authoritative run order (e.g. roles before users);
anything found on disk but missing from it runs last, with a warning, so a
forgotten registration never silently vanishes.

See [`../example/full-sample/database/seeders/mysql`](../example/full-sample/database/seeders/mysql)
for two real, runnable seeders side by side: `UsersSeeder` (plain
`map[string]any` rows via `db.Table(...).Create(...)`) and `PostsSeeder`
(a typed GORM model struct via `db.Create(&posts)`) - both are structs
implementing `Seeder`, living under `SeederPath/<engine>` where gorun's
file scan can find them, registered by name in that package's `registry.go`.

Without a registry set for an engine, `seed run`/`seed list` against that
engine return a clear configuration error instead of panicking.

## Using it from the global CLI: `RunnerPath`

`Config.MySQLSeeders`/`PostgreSQLSeeders` are Go interface values - a
`.gorun/config.yaml` loaded by the global `gorun` binary can never set
them, the same way it can't express any other behavior, only data. Set
`Config.RunnerPath` (`runner_path:` in YAML) to a project-local Go
entrypoint - `gorun setup` scaffolds one at `cmd/gorun-runner/main.go` for
you - and `seed` commands run from the global binary delegate to it via
`go run` instead of failing with "no registry configured". That
entrypoint just calls `gorun.LoadConfigFile` to get the same `Config` the
global binary resolved, attaches real seeders, and calls `gorun.New`
itself - see the generated file's comments.

`gorun setup` also scaffolds `gorun`/`gorun.bat` right next to it - a
shortcut for `go run ./cmd/gorun-runner "$@"`, so any command that needs
your project's own seeders/config (not just `seed run`) can be typed as
`./gorun migrate run` instead of the full `go run` invocation.

`migrate fresh --seed` and `migrate refresh --seed` shell out to `go run
<Config.RunnerPath> seed run ...` the same way, to run seeders after a
rebuild - `--seed` errors immediately with a clear message if
`RunnerPath` isn't set, rather than guessing at a path.
