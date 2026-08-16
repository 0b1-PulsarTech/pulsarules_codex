package simplificationpath

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

// markerPrefix is the text a "// simplification:" comment starts with, once
// its leading slashes and whitespace are trimmed away.
const markerPrefix = "simplification:"

var missingUpgradePathReporter = core.NewReporter(
	"simplification-path", core.SeverityWarning, core.CatSyntax,
)

// Analyzer reports a "// simplification:" comment block that names no
// upgrade path, so a deliberately cut corner cannot be left undocumented.
type Analyzer struct{}

func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

func (a *Analyzer) ID() string   { return "simplification-path" }
func (a *Analyzer) Name() string { return "Simplification path" }
func (a *Analyzer) Description() string {
	return "Reports a simplification comment that names no upgrade path"
}
func (a *Analyzer) Stage() core.StageID     { return core.StageStatic }
func (a *Analyzer) Category() core.Category { return core.CatSyntax }
func (a *Analyzer) Needs() core.Requirements {
	return core.Requirements{}
}

// Analyze reports every "// simplification:" marker that names no upgrade path.
//
// simplification: scans only .go files, unlike textmarkers which also reads .md, though
// markers also exist in opencode-plugin.js (JS shares "//" comments).
// Upgrade path: route non-Go extensions through core.LanguageRegistry.IsCommentLine.
func (a *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	if ctx.Sources == nil {
		return nil
	}
	var findings []core.Finding
	for _, fc := range ctx.ChangedFiles {
		if fc.Extension != ".go" {
			continue
		}
		src, ok := ctx.Sources.Read(fc.Path)
		if !ok {
			continue
		}
		findings = append(findings, scanFile(src, fc.Path)...)
	}
	return findings
}

// scanFile walks src line by line, collecting each "// simplification:"
// comment block (the marker line plus every immediately following comment
// line) and reporting one whose block never states an upgrade-path label.
func scanFile(src []byte, path string) []core.Finding {
	var findings []core.Finding
	lines := strings.Split(string(src), "\n")
	for lineIdx := 0; lineIdx < len(lines); lineIdx++ {
		if !isMarkerLine(lines[lineIdx]) {
			continue
		}
		blockEnd := lineIdx + 1
		for blockEnd < len(lines) && isCommentLine(lines[blockEnd]) {
			blockEnd++
		}
		if !blockNamesUpgradePath(lines[lineIdx:blockEnd]) {
			findings = append(findings, missingUpgradePathReporter.At(
				path,
				lineIdx+1,
				"simplification comment names no upgrade path",
				`add "Upgrade path: ..." or "revisit when ..." to the comment block`,
			))
		}
		lineIdx = blockEnd - 1
	}
	return findings
}

func isCommentLine(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "//")
}

// isMarkerLine reports whether line is a "// simplification:" comment, once
// its leading whitespace and comment slashes are trimmed away.
func isMarkerLine(line string) bool {
	comment, ok := strings.CutPrefix(strings.TrimSpace(line), "//")
	if !ok {
		return false
	}
	return strings.HasPrefix(strings.TrimSpace(comment), markerPrefix)
}

// blockNamesUpgradePath reports whether the block names an upgrade path,
// matched case-insensitively anywhere in it.
//
// why: the comment slashes come off and the lines join with a space, because
// golines wraps prose - "Upgrade path:" split across two lines is still one.
func blockNamesUpgradePath(lines []string) bool {
	words := make([]string, 0, len(lines))
	for _, line := range lines {
		comment, _ := strings.CutPrefix(strings.TrimSpace(line), "//")
		words = append(words, strings.TrimSpace(comment))
	}
	block := strings.ToLower(strings.Join(words, " "))
	return strings.Contains(block, "upgrade path:") || strings.Contains(block, "revisit when")
}
