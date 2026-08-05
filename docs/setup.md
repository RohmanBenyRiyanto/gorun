# Setup

Two ways to bring `gorun` into a project: as a library you wire into your
own `main.go`, or as a globally-installed CLI that discovers a config file.

## As a library

```
go get github.com/RohmanBenyRiyanto/gorun
```

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
    MySQLSeeders: myapp.SeederRegistry{}, // see seeders.md
}

cmd := gorun.New(cfg)
if err := cmd.Run(context.Background(), os.Args); err != nil {
    log.Fatal(err)
}
```

`cmd` is a regular `*cli.Command` (urfave/cli/v3), so it composes normally
if you want to fold it into a bigger CLI of your own. `MigrationPath`/
`SeederPath` (and anything else path-shaped) resolve relative to wherever
the process is run from, not the repo root.

## As a global CLI

```
go install github.com/RohmanBenyRiyanto/gorun/cmd/gorun@latest
```

Run `gorun <command>` from inside any project directory - no `go run
./path/to/main.go` needed. On startup, `gorun` walks up from the current
directory looking for `.gorun/config.yaml`, the same way `git` finds
`.git/`, and loads `Config` from it. See [configuration.md](configuration.md)
for the file format.

Commands that don't touch a database (`help`, `version`, `info`, `app`)
work from anywhere; `db`/`migrate`/`seed`/`table` need a config to be
found first. Pass `--config <path>` to point at a specific file instead of
relying on discovery.

### `gorun setup`

Skip writing the config by hand:

```
gorun setup
```

Prompts interactively when stdin is a terminal (reusing the same prompt UI
the rest of gorun's interactive commands use), and scaffolds
`database/migrations/<engine>` / `database/seeders/<engine>` for whichever
engine(s) you configure. Pass `--yes` plus flags for every question
(`--mysql-host`, `--mysql-user`, `--app-env`, ...) to skip prompting
entirely - for CI, a script, or an AI agent driving setup with no TTY.
`--interactive` forces prompting even when stdin isn't detected as a
terminal; `--force` overwrites an existing config instead of refusing. Run
`gorun setup --help` for the full flag list.

See [`../example/full-sample`](../example/full-sample) for the whole thing
done for real against a local MySQL: `gorun setup` -> `gorun migrate make`
-> `gorun migrate run` -> a real seeder -> `gorun seed run` - every file in
that directory is the actual output of those commands, not hand-written to
look right.

### Already have your own config system?

Most real projects do. See
[`../example/existing-config-sample`](../example/existing-config-sample) -
a thin `.gorun/config.yaml` that only ever duplicates non-secret values
(never the password, which both files read from the same environment
variable), plus a `cmd/gorun-runner` that skips `.gorun/config.yaml`
entirely and builds `gorun.Config` straight from the project's own loader
instead.
