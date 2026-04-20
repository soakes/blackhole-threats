package buildinfo

import (
	"regexp"
	"strings"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

var releaseTagPattern = regexp.MustCompile(`^(v\d+\.\d+\.\d+(?:-rc\.\d+)?)(?:-\d+-g[0-9a-f]+)?(?:-dirty)?$`)

func DisplayVersion() string {
	switch {
	case Version != "" && Version != "dev" && Version != "main" && Version != "master":
		return Version
	case Commit != "" && Commit != "unknown":
		return ShortCommit()
	case Version != "":
		return Version
	default:
		return "unknown"
	}
}

func ShortCommit() string {
	commit := strings.TrimSpace(Commit)
	if commit == "" || commit == "unknown" {
		return "unknown"
	}

	if len(commit) > 7 {
		return commit[:7]
	}

	return commit
}

func TagVersion() string {
	version := strings.TrimSpace(Version)
	if version == "" {
		return "unknown"
	}

	match := releaseTagPattern.FindStringSubmatch(version)
	if len(match) != 2 {
		return "unknown"
	}

	return match[1]
}
