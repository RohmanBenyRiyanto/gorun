// Package setup implements `gorun setup`, the command that makes a
// directory into a gorun project: it writes .gorun/config.yaml and
// scaffolds database/migrations, database/seeders. It runs before any
// gorun.Config exists - that's the whole point - so it lives outside the
// gorun.New command tree and is wired in specially by cmd/gorun.
//
// Two ways to answer its questions: interactively (reusing the prompt
// helpers already in internal/engine) when stdin is a terminal, or
// entirely through flags when it isn't - piped input, CI, or an
// AI agent driving setup with no TTY at all. Every question the wizard
// asks has a matching flag; --yes forces flag-only mode even when a TTY
// is present.
package setup
