package cursor

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/ownedfile"
)

const RulesDir = ".cursor/rules"

const PointerID = "pulsarules-contract"

func WriteRule(projectDir, id, body string) (ownedfile.Outcome, error) {
	path := filepath.Join(projectDir, RulesDir, id+".mdc")
	return ownedfile.WriteOwned(path, []byte(body), disk.ModeFilePrivate, makeRulesDir)
}

func makeRulesDir(dir string) error {
	if err := os.MkdirAll(dir, disk.ModeDirPrivate); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	return nil
}
