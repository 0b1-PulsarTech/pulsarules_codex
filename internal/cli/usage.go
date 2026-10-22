package cli

import (
	_ "embed"
	"fmt"
	"os"
)

//go:embed usage.txt
var usageText string

func Usage() error {
	_, _ = fmt.Fprint(os.Stdout, usageText)
	return nil
}

func IsHelp(command string) bool {
	switch command {
	case "help", "-h", "--help":
		return true
	}
	return false
}
