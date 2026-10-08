package tssyntax

import (
	"os"
	"strconv"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/sandbox"
)

func TestMain(m *testing.M) {
	if RunChild(os.Args, os.Exit) {
		return
	}
	const probe = "probechild"
	if len(os.Args) > 3 && os.Args[1] == probe {
		pid := strconv.Itoa(os.Getpid())
		if err := os.WriteFile(os.Args[2], []byte(pid), 0o600); err != nil {
			os.Exit(2)
		}
		RunChild([]string{os.Args[0], sandbox.ChildArg, os.Args[3]}, os.Exit)
		return
	}
	os.Exit(m.Run())
}
