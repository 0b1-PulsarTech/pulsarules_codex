package banner

import (
	"bytes"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

type Analyzer struct {
	langs    *pass.LanguageRegistry
	reporter pass.Reporter
}

func NewAnalyzer(langs *pass.LanguageRegistry) *Analyzer {
	return &Analyzer{
		langs:    langs,
		reporter: pass.NewReporter("top-of-file", pass.SeverityError, pass.CatSyntax),
	}
}

func (a *Analyzer) ID() string          { return "top-of-file" }
func (a *Analyzer) Stage() pass.StageID { return pass.StageStatic }

func (a *Analyzer) Analyze(ctx *pass.Context) []pass.Finding {
	if a.langs == nil {
		return nil
	}
	reporter := pass.ResolveReporter(a.reporter, ctx)
	// Required header lines (core.ExemptHeaderSpan) are skipped so a hand comment after
	// them is still flagged; a core.IsContentStartLine line ends the scan, and a
	// core.IsGeneratedMarkerLine line before it exempts the file in full.
	extFilter := pass.ExtensionFilter(ctx, a.ID(), defaultExtensions())
	eligible := func(fc pass.FileChange) bool {
		return extFilter(fc) && a.langs.Lookup(fc.Extension) != nil
	}
	return pass.EachChangedFile(ctx, eligible, func(
		fc pass.FileChange, src []byte,
	) []pass.Finding {
		lineNum, ok := leadingCommentLine(a.langs.Lookup(fc.Extension), fc.Extension, src)
		if !ok {
			return nil
		}
		return []pass.Finding{reporter.At(
			fc.Path,
			lineNum,
			"file starts with a comment before the package declaration",
			"remove the comment; no package docstrings",
		)}
	})
}

func defaultExtensions() []string {
	return []string{".go", ".proto", ".sql", ".nix", ".svelte", ".ts", ".dart"}
}

func leadingCommentLine(lang pass.Language, extension string, src []byte) (int, bool) {
	const utf8BOM = "\xef\xbb\xbf"
	lines := strings.Split(string(bytes.TrimPrefix(src, []byte(utf8BOM))), "\n")
	pos := 0
	for pos < len(lines) {
		line := lines[pos]
		if strings.TrimSpace(line) == "" {
			pos++
			continue
		}
		if pass.IsGeneratedMarkerLine(line) {
			return 0, false
		}
		if pass.IsContentStartLine(line) {
			return 0, false
		}
		if span := pass.ExemptHeaderSpan(lines[pos:]); span > 0 {
			pos += span
			continue
		}
		return pos + 1, isComment(lang, extension, line)
	}
	return 0, false
}

func isComment(lang pass.Language, extension, line string) bool {
	switch strings.ToLower(extension) {
	case ".svelte", ".vue":
		// Markup has no "//" or "/* */" comment; only HTML's counts at line 1.
		return strings.HasPrefix(strings.TrimSpace(line), "<!--")
	}
	// A fresh scanner is correct: no earlier line can have left a block open.
	scanner := pass.NewCommentScanner(lang.CommentSyntax())
	return scanner.IsCommentLine(line) && !lang.IsPackageDeclaration(line)
}
