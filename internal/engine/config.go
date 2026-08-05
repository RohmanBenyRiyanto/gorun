package engine

import "time"

// DBConnConfig holds connection settings for one database engine (MySQL or
// PostgreSQL). A zero value means that engine simply isn't configured -
// commands that need it will fail to connect rather than do anything
// destructive with empty credentials.
//
// A few fields are engine-specific and ignored by the other engine:
// Charset/ParseTime/Loc only matter for MySQL, SslMode/TimeZone only for
// PostgreSQL.
type DBConnConfig struct {
	Host, Port     string
	User, Password string
	DatabaseName   string

	// Charset is MySQL-only and defaults to "utf8mb4" if left empty (see
	// getMySQLCharset-equivalent call sites). It does not affect the
	// connection DSN itself - the driver connection always negotiates
	// utf8mb4/parseTime=true/loc=Local regardless of these fields, matching
	// the original tool's behavior. Charset/Loc do drive CREATE DATABASE
	// statements and status output.
	Charset   string
	ParseTime bool
	Loc       string

	// SslMode and TimeZone are PostgreSQL-only and go straight into the
	// connection DSN, so an empty value here becomes `sslmode=` /
	// `TimeZone=` on the wire. New fills in "disable" / "UTC" when left
	// zero - see New's default handling.
	SslMode  string
	TimeZone string

	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration

	// MigrationPath and SeederPath are joined with "mysql"/"postgresql" to
	// locate migration and seeder files (see MigrationManager and
	// SeederManager). There's no invented fallback for an empty value here
	// - it resolves to a bare "mysql" or "postgresql" directory relative to
	// the current working directory, exactly like the tool this was ported
	// from. Set these explicitly unless that's genuinely what you want.
	MigrationPath string
	SeederPath    string
}

// IsConfigured reports whether this engine has real connection settings.
// Host is the one field every genuine config sets and nothing defaults it
// for you, so an empty Host is the signal - consistent with this type's
// own zero-value contract above.
func (c DBConnConfig) IsConfigured() bool {
	return c.Host != ""
}

// Config is the one piece of shared state every gorun command reads from.
// Build one and pass it to New.
type Config struct {
	// MySQL and PostgreSQL are independent - set whichever engine(s) your
	// project actually uses. Leaving one at its zero value just means
	// commands targeting that engine will fail to connect.
	MySQL      DBConnConfig
	PostgreSQL DBConnConfig

	// MultiDB must be explicitly true before any command will let you
	// choose between MySQL and PostgreSQL at runtime (interactively, or
	// via --type) when both happen to be configured. Without it, having
	// two engines configured is treated as a project misconfiguration
	// (see DatabaseManager.resolveConfigured) rather than something a
	// command silently offers a picker for - a project that only ever
	// meant to use one engine shouldn't have every db/migrate/seed/table
	// command quietly capable of touching the other one too. Leave it
	// false for the common single-database case; it's ignored entirely
	// when zero or one engine is configured.
	MultiDB bool

	// AppEnv gates the "don't seed production by accident" guard in `seed
	// run` (see isProdEnv): "prod" or "production" blocks the run unless
	// --force is passed. Leave it empty and the guard never triggers.
	AppEnv string

	// MySQLSeeders and PostgreSQLSeeders supply the seeder registry that
	// `seed run` / `seed list` use to discover and order your project's
	// seeders. gorun ships no seeders of its own - only the machinery to
	// run them - so leave the engine you don't use as nil, and implement
	// SeederRegistry for the one(s) you do. A command that needs a
	// registry you didn't set reports a configuration error rather than
	// panicking.
	MySQLSeeders      SeederRegistry
	PostgreSQLSeeders SeederRegistry

	// Name is the program name shown in help/usage output. Defaults to
	// "gorun" if left empty.
	Name string

	// Usage is the one-line banner shown alongside Name in help output.
	// Defaults to a generic description if left empty.
	Usage string

	// ServerEntrypoint is the Go file `app build`/`app serve` compile or
	// run - relative to the project root. Defaults to
	// "cmd/server/main.go" if left empty, matching the layout the tool
	// this was ported from assumed unconditionally.
	ServerEntrypoint string

	// RunnerPath is only read by the globally-installed gorun binary
	// (cmd/gorun), never by the library. It's the project-local Go
	// entrypoint - typically scaffolded by `gorun setup` - that `seed`
	// commands delegate to via `go run` when MySQLSeeders/PostgreSQLSeeders
	// are nil, which is always true for a Config loaded from
	// .gorun/config.yaml (YAML can't express a Go interface
	// implementation). That delegate is expected to build its own Config
	// with real seeders and call New itself - see the gorun README's "CLI
	// usage" section. Leave empty if you only ever use gorun as a
	// library, or don't need `seed` from the global binary.
	RunnerPath string
}
