// Package gorun is a Laravel-artisan-style command-line toolkit for Go
// projects: database/table management, SQL migrations, seeders, and a
// handful of application lifecycle commands (build/serve/test/clean/
// install), all wired up behind one urfave/cli/v3 command tree.
//
// Build a Config, pass it to New, and run the result:
//
//	cmd := gorun.New(gorun.Config{
//		MySQL: gorun.DBConnConfig{
//			Host: "127.0.0.1", Port: "3306",
//			User: "root", DatabaseName: "myapp",
//			MigrationPath: "database/migrations",
//			SeederPath:    "database/seeders",
//		},
//	})
//	if err := cmd.Run(context.Background(), os.Args); err != nil {
//		log.Fatal(err)
//	}
//
// See README.md and example/full-sample for a complete walkthrough,
// including how to wire up your own seeders via SeederRegistry.
package gorun
