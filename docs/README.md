# gorun documentation

Deeper reference material for the `gorun` package. Start here, then jump
to whichever page matches what you're doing.

| Page | What's in it |
|---|---|
| [setup.md](setup.md) | Installing `gorun`, `gorun setup` wizard walkthrough, how config discovery works (`.gorun/config.yaml`) |
| [configuration.md](configuration.md) | Every `Config` field, the YAML file format, single- vs. multi-database engine resolution, the pre-destructive-action confirmation |
| [commands.md](commands.md) | Every command group and subcommand in detail: `db`, `table`, `migrate`, `seed`, `app` |
| [seeders.md](seeders.md) | The `Seeder`/`SeederRegistry` interfaces, run ordering, and `RunnerPath` delegation for the global CLI |

For runnable, end-to-end setups instead of reference docs, see
[`../example/`](../example/):

- [`full-sample`](../example/full-sample) - `gorun setup` -> migrate ->
  seed, done for real against a local MySQL, using the global binary.
- [`existing-config-sample`](../example/existing-config-sample) -
  integrating `gorun` into a project that already has its own config
  system.
- [`single-db`](../example/single-db) / [`multi-db`](../example/multi-db) -
  minimal single-engine and dual-engine (`MultiDB: true`) setups.

This directory is a work in progress - more pages will be added over time.
