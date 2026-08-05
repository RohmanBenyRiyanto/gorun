package setup

// dbAnswers holds one database engine's connection settings as collected
// by either the wizard or flags. Configure is false when the user (or
// the caller of a non-interactive run) never set anything for this
// engine - the resulting config simply omits that section, same as
// leaving DBConnConfig at its zero value.
type dbAnswers struct {
	Configure bool

	Host, Port, User, Password, DatabaseName string
	MigrationPath, SeederPath                string
}

// runnerPath is the one place gorun-runner ever lives - not
// user-configurable through setup, deliberately, in the same spirit as
// Laravel's own fixed conventions (database/migrations, database/seeders):
// one obvious answer beats one more flag to learn. Anyone who genuinely
// needs a different layout can still hand-edit .gorun/config.yaml
// afterward.
const runnerPath = "cmd/gorun-runner"

// answers is everything setup needs to write a config file, regardless of
// how it was collected.
type answers struct {
	Name, Usage, AppEnv string
	Extends             string
	MultiDB             bool
	MySQL, PostgreSQL   dbAnswers
}

// wantsRunner reports whether setup should scaffold cmd/gorun-runner -
// only meaningful once at least one engine is actually configured.
func (a answers) wantsRunner() bool {
	return a.MySQL.Configure || a.PostgreSQL.Configure
}
