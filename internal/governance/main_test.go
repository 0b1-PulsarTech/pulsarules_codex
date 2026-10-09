package governance

import (
	"os"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/tssyntax"
)

func TestMain(m *testing.M) {
	if tssyntax.RunChild(os.Args, os.Exit) {
		return
	}
	os.Exit(m.Run())
}
