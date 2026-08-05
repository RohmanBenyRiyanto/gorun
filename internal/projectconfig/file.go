package projectconfig

// File is the on-disk shape of a project's .gorun/config.yaml. It mirrors
// gorun.Config/gorun.DBConnConfig field for field, plus a couple of
// concerns that only exist once gorun is driven from a file instead of
// built in Go code: Extends and RunnerPath.
type File struct {
	// Extends points at another config file - absolute, ~-relative, or
	// relative to the directory this file lives in - whose values seed
	// this file's own defaults before anything below is applied. This is
	// how several projects share settings (e.g. one local database's
	// credentials) without literally becoming the same project the way a
	// symlinked config would: every project still keeps its own real,
	// readable, git-diffable file.
	Extends string `yaml:"extends,omitempty"`

	Name  string `yaml:"name,omitempty"`
	Usage string `yaml:"usage,omitempty"`

	// MultiDB must be explicitly true before db/migrate/seed/table
	// commands will let you pick between mysql and postgresql at runtime
	// when both are configured below - see gorun.Config.MultiDB for why
	// two configured engines without this set is treated as a
	// misconfiguration rather than an implicit feature.
	MultiDB bool `yaml:"multi_db,omitempty"`

	// AppEnv gates seed run's "don't seed production by accident" guard -
	// see gorun.Config.AppEnv.
	AppEnv string `yaml:"app_env,omitempty"`

	// RunnerPath points at a project-local Go entrypoint - typically
	// scaffolded by `gorun setup` - that `seed` commands run via `go run`
	// when this project's seeders can only be expressed as real Go code
	// (a SeederRegistry implementation), which a YAML file can't contain.
	// See gorun.Config.RunnerPath.
	RunnerPath string `yaml:"runner_path,omitempty"`

	// ServerEntrypoint is the Go file `app build`/`app serve` compile or
	// run, relative to the project root. See gorun.Config.ServerEntrypoint.
	ServerEntrypoint string `yaml:"server_entrypoint,omitempty"`

	MySQL      *DBSection `yaml:"mysql,omitempty"`
	PostgreSQL *DBSection `yaml:"postgresql,omitempty"`
}

// DBSection is one database engine's connection settings as written in
// YAML. ConnMaxLifetime is a duration string ("5m", "30s") rather than
// gorun.DBConnConfig's time.Duration, since YAML has no native duration
// type - see parseDuration in load.go. ParseTime is a *bool rather than
// bool so that an explicit `parse_time: false` in a child file can be told
// apart from "not set, inherit from extends" during merging.
type DBSection struct {
	Host         string `yaml:"host,omitempty"`
	Port         string `yaml:"port,omitempty"`
	User         string `yaml:"user,omitempty"`
	Password     string `yaml:"password,omitempty"`
	DatabaseName string `yaml:"database_name,omitempty"`

	// Charset and Loc are MySQL-only, SslMode and TimeZone are
	// PostgreSQL-only - see gorun.DBConnConfig's doc comment for why.
	Charset   string `yaml:"charset,omitempty"`
	ParseTime *bool  `yaml:"parse_time,omitempty"`
	Loc       string `yaml:"loc,omitempty"`

	SslMode  string `yaml:"ssl_mode,omitempty"`
	TimeZone string `yaml:"time_zone,omitempty"`

	MaxOpenConns    int    `yaml:"max_open_conns,omitempty"`
	MaxIdleConns    int    `yaml:"max_idle_conns,omitempty"`
	ConnMaxLifetime string `yaml:"conn_max_lifetime,omitempty"`

	MigrationPath string `yaml:"migration_path,omitempty"`
	SeederPath    string `yaml:"seeder_path,omitempty"`
}
