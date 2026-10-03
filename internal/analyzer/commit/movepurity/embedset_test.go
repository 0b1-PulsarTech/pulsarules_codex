package movepurity

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

func assetRow(name, directive, oldFile, newFile string) renameCase {
	const embedderGo = "internal/assets/embed.go"
	return renameCase{
		name: name,
		head: map[string]string{
			embedderGo: embedSrc("assets", directive),
			oldFile:    "asset\n",
		},
		staged: map[string]string{
			embedderGo: embedSrc("assets", directive),
			newFile:    "asset\n",
		},
		renames: []pass.Rename{pair(oldFile, newFile)},
	}
}

func TestEmbedSet_StaysPure(t *testing.T) {
	t.Parallel()

	const static = "//go:embed static"
	testCases := []renameCase{
		accepted(assetRow(
			"unrelated directory",
			static, "internal/other/a.png", "internal/elsewhere/a.png",
		)),
		accepted(assetRow(
			"file beside the embedder that no pattern names",
			"//go:embed public.txt",
			"internal/assets/other.png",
			"internal/assets/img/other.png",
		)),
		accepted(assetRow(
			"hidden file under a plain directory pattern",
			static, "internal/assets/static/.keep", "internal/other/keep",
		)),
		accepted(assetRow(
			"file under a hidden directory below a plain directory pattern",
			static, "internal/assets/static/_gen/a.txt", "internal/other/a.txt",
		)),
	}
	runRenameCases(t, testCases)
}
