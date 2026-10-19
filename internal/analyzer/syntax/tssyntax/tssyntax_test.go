package tssyntax

import (
	"os"
	"strconv"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

const pidFileEnv = "TSSYNTAX_PIDFILE"

func TestMain(m *testing.M) {
	if path := os.Getenv(pidFileEnv); path != "" {
		if err := os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())), 0o600); err != nil {
			os.Exit(2)
		}
	}
	if RunChild(os.Args, os.Exit) {
		return
	}
	os.Exit(m.Run())
}

func withContent(sources []syntax.Source, read func(string) ([]byte, bool)) []syntax.Source {
	filled := make([]syntax.Source, 0, len(sources))
	for _, source := range sources {
		content, ok := read(source.Path)
		if !ok {
			continue
		}
		source.Content = content
		filled = append(filled, source)
	}
	return filled
}
