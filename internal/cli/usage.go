package cli

import (
	_ "embed"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/command"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/flags"
)

//go:embed usage.txt
var usageProse string

func Usage() error {
	return writeUsage(os.Stdout)
}

func writeUsage(out io.Writer) error {
	synopsis, err := synopsisLines()
	if err != nil {
		return err
	}
	var text strings.Builder
	text.WriteString("pulsarules_cli - render and manage engineering-standards skills\n\nUsage:\n")
	for _, line := range synopsis {
		text.WriteString("  pulsarules_cli " + line + "\n")
	}
	text.WriteString("\n" + usageProse)
	if _, err = io.WriteString(out, text.String()); err != nil {
		return fmt.Errorf("write usage: %w", err)
	}
	return nil
}

func synopsisLines() ([]string, error) {
	lines := make([]string, 0, len(table())+1)
	for _, candidate := range table() {
		runner := candidate.build(flags.Environment{})
		flagSet := flag.NewFlagSet(candidate.name, flag.ContinueOnError)
		var root, logLevel string
		flags.BindCommon(flagSet, &root, &logLevel, flags.Environment{})
		runner.Bind(flagSet)
		terms, err := command.Synopsis(flagSet, runner.Terms())
		if err != nil {
			return nil, fmt.Errorf("synopsis of %s: %w", candidate.name, err)
		}
		lines = append(lines, strings.TrimSpace(candidate.name+" "+terms))
	}
	return append(lines, "version"), nil
}

func IsHelp(command string) bool {
	switch command {
	case "help", "-h", "--help":
		return true
	}
	return false
}
