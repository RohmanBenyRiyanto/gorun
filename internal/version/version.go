package version

import (
	"regexp"
	"runtime/debug"
)

var version string
var pseudoVersionSuffix = regexp.MustCompile(`\d{14}-[0-9a-f]{12}`)

func Get() string {
	if version != "" {
		return version
	}

	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}

	if bi.Main.Version != "" && bi.Main.Version != "(devel)" && !pseudoVersionSuffix.MatchString(bi.Main.Version) {
		return bi.Main.Version
	}

	var revision string
	var dirty bool
	for _, setting := range bi.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			dirty = setting.Value == "true"
		}
	}

	if revision == "" {
		return "dev"
	}
	if len(revision) > 7 {
		revision = revision[:7]
	}
	if dirty {
		return "dev-" + revision + "-dirty"
	}
	return "dev-" + revision
}
