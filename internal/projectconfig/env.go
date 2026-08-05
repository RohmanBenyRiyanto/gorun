package projectconfig

import (
	"os"
	"regexp"
)

// envRefPattern matches ${VAR_NAME} references inside a config file, the
// same syntax docker-compose and most .env-adjacent tooling uses.
var envRefPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

// interpolateEnv replaces every ${VAR} in raw with the value of the
// environment variable VAR (empty string if it isn't set). Applied to the
// whole file before YAML parsing, so it works uniformly across every
// field - most importantly database passwords, which should live in the
// environment rather than get committed in plain text alongside the rest
// of a project's config.
func interpolateEnv(raw []byte) []byte {
	return envRefPattern.ReplaceAllFunc(raw, func(match []byte) []byte {
		name := envRefPattern.FindSubmatch(match)[1]
		return []byte(os.Getenv(string(name)))
	})
}
