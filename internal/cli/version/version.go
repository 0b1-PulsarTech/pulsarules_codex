package version

import (
	"fmt"
	"os"
	"runtime/debug"
	"strings"
)

const fallbackVersion = "0.2.0"

func IsVersion(command string) bool {
	switch command {
	case "version", "-v", "--version":
		return true
	}
	return false
}

func PrintVersion() {
	_, _ = fmt.Fprintf(os.Stdout, "pulsarules_cli %s\n", current())
}

func current() string {
	build, _ := debug.ReadBuildInfo()
	return formatVersion(build)
}

func formatVersion(build *debug.BuildInfo) string {
	// simplification: vcs.revision is trusted as-is; a worktree nested in its
	// own main checkout carries the main checkout's revision. Upgrade path:
	// hash embedded knowledge instead of reading the stamp.
	if build == nil {
		return fallbackVersion
	}
	revision := buildRevision(build)
	version := build.Main.Version
	if version == "" || version == "(devel)" {
		// A go run build has no vcs.revision, so it reports the cut release.
		if revision == "" {
			return fallbackVersion
		}
		return "devel+" + revision
	}
	// A pseudo-version already ends in the revision it was derived from.
	if revision == "" || strings.Contains(version, revision) {
		return version
	}
	return version + "+" + revision
}

func buildRevision(build *debug.BuildInfo) string {
	const shortRevisionLen = 12
	for _, setting := range build.Settings {
		if setting.Key != "vcs.revision" {
			continue
		}
		if len(setting.Value) > shortRevisionLen {
			return setting.Value[:shortRevisionLen]
		}
		return setting.Value
	}
	return ""
}
