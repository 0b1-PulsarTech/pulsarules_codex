package cliopts

import (
	"fmt"
	"os"
	"runtime/debug"
	"strings"
)

// fallbackVersion names a build the toolchain stamped nothing into (`go run`).
// why: it tracks the last cut tag, so an unstamped build names the release it
// was built from rather than freezing at whatever it said when written.
const fallbackVersion = "0.7.2"

// shortRevisionLen trims a stamped commit to the length a person reads.
const shortRevisionLen = 12

// IsVersion reports whether the command requests the version banner.
func IsVersion(command string) bool {
	switch command {
	case "version", "-v", "--version":
		return true
	}
	return false
}

func PrintVersion() {
	_, _ = fmt.Fprintf(os.Stdout, "pulsarules_cli %s\n", Version())
}

// Version reports the running binary's version, preferring what the toolchain
// stamped into it over any literal in this source.
func Version() string {
	build, _ := debug.ReadBuildInfo()
	return formatVersion(build)
}

// formatVersion renders build as a version string: the stamped module version,
// "devel" plus the commit for a local build, or the bare fallback when build is
// nil or carries neither.
func formatVersion(build *debug.BuildInfo) string {
	// simplification: the stamped vcs.revision is trusted. Go derives it from
	// the build directory, so a build made from a worktree nested inside its own
	// main checkout carries the MAIN checkout's revision. Upgrade path: hash the
	// embedded knowledge instead of reading the stamp.
	if build == nil {
		return fallbackVersion
	}
	revision := buildRevision(build)
	version := build.Main.Version
	if version == "" || version == "(devel)" {
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
