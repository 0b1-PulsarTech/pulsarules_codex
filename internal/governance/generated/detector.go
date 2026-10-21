package generated

import (
	"path/filepath"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/lexer"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/source"
)

type Detector struct {
	provider  source.Provider
	atlasDirs map[string]bool
	known     map[string]bool
}

func NewDetector(provider source.Provider) *Detector {
	return &Detector{
		provider:  provider,
		atlasDirs: map[string]bool{},
		known:     map[string]bool{},
	}
}

func (detector *Detector) IsGenerated(path string) bool {
	if generated, ok := detector.known[path]; ok {
		return generated
	}
	generated := detector.inAtlasDir(path)
	if !generated && detector.provider != nil {
		if src, found := detector.provider.Read(path); found {
			generated = lexer.IsGeneratedSource(src)
		}
	}
	detector.known[path] = generated
	return generated
}

func (detector *Detector) Mark(files []pass.FileChange) {
	for idx := range files {
		files[idx].Generated = detector.IsGenerated(files[idx].Path)
	}
}

func (detector *Detector) Split(findings []finding.Finding) ([]finding.Finding, int) {
	kept := make([]finding.Finding, 0, len(findings))
	suppressed := 0
	for _, finding := range findings {
		// A finding with no file (a commit-message rule) belongs to no file and is never
		// suppressed; a generated file's are, because the next codegen run undoes any edit.
		if finding.File != "" && detector.IsGenerated(finding.File) {
			suppressed++
			continue
		}
		kept = append(kept, finding)
	}
	return kept, suppressed
}

func (detector *Detector) inAtlasDir(path string) bool {
	if detector.provider == nil || !strings.EqualFold(filepath.Ext(path), ".sql") {
		return false
	}
	dir := filepath.ToSlash(filepath.Dir(path))
	generated, ok := detector.atlasDirs[dir]
	if !ok {
		_, generated = detector.provider.Read(filepath.ToSlash(filepath.Join(dir, "atlas.sum")))
		detector.atlasDirs[dir] = generated
	}
	return generated
}
