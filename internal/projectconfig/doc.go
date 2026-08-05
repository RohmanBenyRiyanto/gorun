// Package projectconfig loads the on-disk config a globally-installed
// gorun binary needs, since there's no consumer main.go around to build a
// gorun.Config in Go code. It owns discovery (walking up from the working
// directory for a .gorun/config.yaml, the same way git finds .git/), the
// YAML schema for that file, env-variable interpolation for secrets, and
// the extends chain that lets several projects share settings without
// becoming the same project.
package projectconfig
