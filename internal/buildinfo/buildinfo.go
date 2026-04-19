package buildinfo

import "strings"

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

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
