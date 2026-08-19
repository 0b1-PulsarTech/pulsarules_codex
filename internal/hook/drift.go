package hook

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// emitKnowledgeDrift warns when the standards on disk differ from the ones this
// binary embeds: the installed binary is then behind the source it was built
// from, rendering old skills and running old checks while reporting success. No
// session gate of its own - the only caller already fires once per session.
func (d *Dispatcher) emitKnowledgeDrift() {
	projectDir := d.resolveProjectDir()
	if projectDir == "" {
		return
	}
	if notice, found := knowledgeDriftNotice(projectDir); found {
		d.emitOutput("SessionStart", notice)
	}
}

// knowledgeDriftNotice compares projectDir's standards against the embedded ones,
// reporting the notice and whether there is anything to say.
// why: no standards tree is a legitimate install target with nothing to compare,
// so it is SILENT. A tree that cannot be READ says so instead - collapsing both
// into silence would report "up to date" forever after a permission error.
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
		"Could not verify whether the installed binary matches the knowledge tree: %v.", err,
	)
}
