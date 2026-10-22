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

func (detector *Detector) isGenerated(path string) bool {
	if marked, ok := detector.known[path]; ok {
		return marked
	}
	marked := detector.inAtlasDir(path)
	if !marked && detector.provider != nil {
		if src, found := detector.provider.Read(path); found {
			marked = lexer.IsGeneratedSource(src)
		}
	}
	detector.known[path] = marked
	return marked
}

func (detector *Detector) Mark(files []pass.FileChange) {
	for idx := range files {
		files[idx].Generated = detector.isGenerated(files[idx].Path)
	}
}

func (detector *Detector) Split(findings []finding.Finding) ([]finding.Finding, int) {
	kept := make([]finding.Finding, 0, len(findings))
	suppressed := 0
	for _, hit := range findings {
		// A finding with no file (a commit-message rule) belongs to no file and is never
		// suppressed; a generated file's are, because the next codegen run undoes any edit.
		if hit.File != "" && detector.isGenerated(hit.File) {
			suppressed++
			continue
		}
		kept = append(kept, hit)
	}
	return kept, suppressed
}

func (detector *Detector) inAtlasDir(path string) bool {
	if detector.provider == nil || !strings.EqualFold(filepath.Ext(path), ".sql") {
		return false
	}
	dir := filepath.ToSlash(filepath.Dir(path))
	hasSum, ok := detector.atlasDirs[dir]
	if !ok {
		_, hasSum = detector.provider.Read(filepath.ToSlash(filepath.Join(dir, "atlas.sum")))
		detector.atlasDirs[dir] = hasSum
	}
	return hasSum
}
