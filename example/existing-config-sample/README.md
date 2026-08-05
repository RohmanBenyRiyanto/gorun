# gorun + a project's own existing config

Most real projects already have a config system before they ever meet
gorun - an `app.yaml` (or similarly-named) file for non-secret settings,
an environment variable for the database password specifically, resolved
once at startup into a `configs.Get()`-style function. This sample is a
small stand-in for that (`internal/config`), answering one question: **do
you have to duplicate your connection settings into `.gorun/config.yaml`
too?**

Short answer: barely, and never the secret part. Two things happen here
at once, both hitting the same database, verified by actually running
them:

## 1. Non-secret settings: a thin, literal adapter

[`.gorun/config.yaml`](.gorun/config.yaml) - used by `gorun db status`,
`gorun migrate run`, `gorun table list`, anything invoked directly from
the global binary - just copies `host`/`port`/`user`/`database_name` from
[`app.yaml`](app.yaml). That's a real, accepted duplication, but a small
and low-stakes one: these values aren't secret and rarely change, so one
extra literal copy in a file gorun itself needs is simpler than routing
everything through environment variables just to avoid it.

The password is the one field handled differently: `.gorun/config.yaml`
references `${DB_MYSQL_PASSWORD}` - the *exact same* environment variable
[`internal/config/config.go`](internal/config/config.go)'s `MySQLPassword()`
reads. Neither file ever contains the actual secret; both point at the
same one source.

## 2. `seed run`: zero duplication at all

[`cmd/gorun-runner/main.go`](cmd/gorun-runner/main.go) - what `gorun seed
run` delegates to (see `runner_path` in `.gorun/config.yaml`) - doesn't
call `gorun.LoadConfigFile` at all. It calls this project's own
`config.Load("app.yaml")`, the same function any other package in a real
app would call, and maps the result into a `gorun.Config` by hand:

```go
appCfg, _ := config.Load("app.yaml")

cfg := gorun.Config{
    MySQL: gorun.DBConnConfig{
        Host:         appCfg.Database.MySQL.Host,
        Port:         appCfg.Database.MySQL.Port,
        User:         appCfg.Database.MySQL.User,
        Password:     appCfg.MySQLPassword(),
        DatabaseName: appCfg.Database.MySQL.DBName,
        // ...
    },
    MySQLSeeders: mysqlseeders.Registry{}, // database/seeders/mysql/registry.go
}
```

`Registry` lives in `database/seeders/mysql/registry.go`, next to the
seeder structs it registers - not in `main.go` - so adding a seeder never
touches this file.

No `.gorun/config.yaml` involved in this path whatsoever - if your
project already has a config loader you trust, this is the cleaner
option wherever gorun is driven from your own Go code (a runner, or
gorun used as a straight library per `../main.go`) rather than the
prebuilt global binary.

## Try it

```
mysql -uroot -e "CREATE DATABASE gorun_existing_config_sample"
cd example/existing-config-sample
export DB_MYSQL_PASSWORD=            # empty - local root has no password
gorun db status mysql                # pattern 1: .gorun/config.yaml
gorun migrate run --force            # pattern 1: real products table created
gorun seed run                       # pattern 2: config.Load(), 2 real rows inserted
```

(`gorun` here means the real globally-installed binary -
`go install github.com/RohmanBenyRiyanto/gorun/cmd/gorun@latest` - not `go
run`, since the whole point is showing this working the way an actual
consumer project would use it.)
