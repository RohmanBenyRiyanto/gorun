# Configuration

```go
type Config struct {
    MySQL      DBConnConfig
    PostgreSQL DBConnConfig
    MultiDB    bool

    AppEnv string

    MySQLSeeders      SeederRegistry
    PostgreSQLSeeders SeederRegistry

    Name  string
    Usage string

    ServerEntrypoint string
    RunnerPath       string
}
```

Leave `MySQL`/`PostgreSQL` unset for whichever engine you don't use -
commands targeting it will fail to connect rather than do anything
destructive. `gorun.New` fills in a few zero-value defaults: `Name` ->
`"gorun"`, `Usage` -> a generic description, `MySQL.Charset` ->
`"utf8mb4"`, `MySQL.Loc` -> `"Local"`, `PostgreSQL.SslMode` ->
`"disable"`, `PostgreSQL.TimeZone` -> `"UTC"`, `ServerEntrypoint` ->
`"cmd/server/main.go"`.

`AppEnv` gates one thing: `seed run` refuses to run when `AppEnv` is
`"prod"` or `"production"`, unless you pass `--force`. It's not a general
environment concept - just that one guard's input.

## The YAML file (`.gorun/config.yaml`)

Same fields as `Config` above, in YAML, with `${VAR}` interpolated from
the environment (so passwords don't need to sit in the file itself), plus
an `extends:` field for sharing settings across several projects without
making them the same project. See
[`../example/full-sample/.gorun/config.yaml`](../example/full-sample/.gorun/config.yaml)
for a filled-in example, and [setup.md](setup.md) for how it gets written
in the first place.

## Single vs. multiple database engines

`db`/`migrate`/`seed`/`table` commands need to know which engine to talk
to. That resolution is automatic and safe by default:

- **Neither configured**: the command refuses to run with a clear error.
- **Exactly one configured**: it's used automatically - no `--type` flag,
  no prompt, no ceremony. This is the common case.
- **Both configured**: commands refuse to guess *unless* `MultiDB` is
  `true`, in which case they prompt (or accept `--type mysql`/`--type
  postgresql`) to pick one. Two engines configured without `MultiDB` set
  is treated as a project misconfiguration, not an implicit feature -
  otherwise every command on a project that only meant to use one engine
  would quietly be capable of touching the other one too. An explicit
  `--type` flag always works regardless of `MultiDB`, since typing it on
  the command line is itself a deliberate, unambiguous choice.

`gorun setup` enforces the same rule: configuring both engines without
also confirming multi-database intent (the wizard asks; `--yes` mode
needs `--multi-db`) fails setup outright rather than writing a config that
every later command would just refuse to use.

See [`../example/single-db`](../example/single-db) and
[`../example/multi-db`](../example/multi-db) for both shapes configured
for real.

## Which database, once the engine's resolved

Commands that then need one specific database on that server
(`migrate`/`seed run`/`table list`/`table truncate`) use
`DBConnConfig.DatabaseName` directly - the way Laravel's `DB_DATABASE`
works, one project means one configured database, decided once, not
re-picked from a list on every command. They only fall back to listing
every database on the server and prompting when `DatabaseName` is left
empty; `--database <name>` always overrides both.

## One last checkpoint before anything destructive

`db drop`, `db truncate`, `table drop`, `table truncate`, and `migrate
reset`/`rollback`/`fresh` all print the resolved target - engine,
host:port, database - immediately before their `y/N` confirmation (skip
both with `--force`). With engine and database resolved automatically in
the common case, this confirmation is the one place left to notice
"that's not the server I meant" before it's too late.
