package version

import (
	"fmt"
	"os"
)

const version = "0.2.0"

func IsVersion(command string) bool {
	switch command {
	case "version", "-v", "--version":
		return true
	}
	return false
}

func PrintVersion() {
	_, _ = fmt.Fprintf(os.Stdout, "pulsarules_cli %s\n", version)
}
