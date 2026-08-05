package setup

import (
	"fmt"

	"github.com/urfave/cli/v3"
)

// answersFromFlags builds answers entirely from CLI flags - the path
// used when --yes is passed or stdin isn't a terminal (CI, a script, an
// AI agent driving setup with no TTY). Every question the interactive
// wizard asks has a flag equivalent here; nothing is silently defaulted
// beyond what the flags themselves default to.
func answersFromFlags(cmd *cli.Command) (answers, error) {
	a := answers{
		Name:    cmd.String("name"),
		Usage:   cmd.String("usage"),
		AppEnv:  cmd.String("app-env"),
		Extends: cmd.String("extends"),
		MultiDB: cmd.Bool("multi-db"),
	}

	a.MySQL = dbAnswersFromFlags(cmd, "mysql")
	a.PostgreSQL = dbAnswersFromFlags(cmd, "postgresql")

	if !a.MySQL.Configure && !a.PostgreSQL.Configure && a.Extends == "" {
		return answers{}, fmt.Errorf("setup --yes needs at least one of --mysql-host, --postgresql-host, or --extends - nothing to configure otherwise")
	}
	if a.MySQL.Configure && a.PostgreSQL.Configure && !a.MultiDB {
		return answers{}, fmt.Errorf("both --mysql-host and --postgresql-host were given - pass --multi-db too to confirm this project deliberately uses both engines, otherwise configure only the one you use")
	}

	return a, nil
}

// dbAnswersFromFlags reads one engine's flags under the given prefix
// ("mysql" or "postgresql"). Configure is true only if a host was given -
// that's the signal this engine is actually wanted, since every other
// field has a usable default.
func dbAnswersFromFlags(cmd *cli.Command, prefix string) dbAnswers {
	host := cmd.String(prefix + "-host")
	return dbAnswers{
		Configure:     host != "",
		Host:          host,
		Port:          cmd.String(prefix + "-port"),
		User:          cmd.String(prefix + "-user"),
		Password:      cmd.String(prefix + "-password"),
		DatabaseName:  cmd.String(prefix + "-database"),
		MigrationPath: cmd.String(prefix + "-migration-path"),
		SeederPath:    cmd.String(prefix + "-seeder-path"),
	}
}
