package cliopts

import (
	"fmt"
	"os"
)

// version is the installer's release version. Bump it whenever the rendered
// skills, the hook wiring, or the MCP/opencode install behaviour change so
// existing projects know to re-pull. Pre-1.0: the API may still shift.
const version = "0.2.0"

// IsVersion reports whether the command requests the version banner.
func IsVersion(command string) bool {
	switch command {
	case "version", "-v", "--version":
		return true
	}
	return false
}

// PrintVersion writes the version banner to stdout.
func PrintVersion() {
	_, _ = fmt.Fprintf(os.Stdout, "pulsarules_cli %s\n", version)
}
