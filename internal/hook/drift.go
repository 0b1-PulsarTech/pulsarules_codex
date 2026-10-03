package hook

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/printable"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func (disp *Dispatcher) emitKnowledgeDrift() {
	projectDir := disp.resolveProjectDir()
	if projectDir == "" {
		return
	}
	if notice, found := knowledgeDriftNotice(projectDir); found {
		disp.emitOutput("SessionStart", notice)
	}
}

func knowledgeDriftNotice(projectDir string) (string, bool) {
	onDisk, err := knowledge.FingerprintDir(filepath.Join(projectDir, "knowledge"))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", false
		}
		return unverifiableNotice(err), true
	}
	embedded, err := knowledge.Fingerprint()
	if err != nil {
		return unverifiableNotice(err), true
	}
	if embedded == onDisk {
		return "", false
	}
	return "The knowledge tree on disk differs from the standards this binary " +
		"embeds, so the skills it renders and the checks it runs are behind the " +
		"source. Rebuild and re-run the installer.", true
}

func unverifiableNotice(err error) string {
	return fmt.Sprintf(
		"Could not verify whether the installed binary matches the knowledge tree: %s.",
		printable.Line(err.Error()),
	)
}
