package singleletter

import (
	"fmt"
	"unicode"
	"unicode/utf8"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/syntaxfiles"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax"
)

type Analyzer struct{}

func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

func (analyzer *Analyzer) ID() string          { return "single-letter-name" }
func (analyzer *Analyzer) Stage() core.StageID { return core.StageAST }

func (analyzer *Analyzer) reporter() core.Reporter {
	return core.NewReporter(analyzer.ID(), core.SeverityWarning, core.CatAST)
}

func (analyzer *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	cfg := settings{
		reporter:     analyzer.reporter().Resolved(ctx),
		exportedOnly: ctx.Params(analyzer.ID()).Bool("exported_only", false),
	}
	return syntaxfiles.Each(ctx, func(fc core.FileChange, file *syntax.File) []core.Finding {
		if fc.Generated {
			return nil
		}
		return checkFile(fc, file, cfg)
	})
}

type settings struct {
	reporter     core.Reporter
	exportedOnly bool
}

func checkFile(fc core.FileChange, file *syntax.File, cfg settings) []core.Finding {
	findings := make([]core.Finding, 0, len(file.Names))
	for _, name := range file.Names {
		if name.Exempt || !isSingleLetter(name.Name) || (cfg.exportedOnly && !name.Exported) {
			continue
		}
		findings = append(findings, cfg.reporter.At(
			fc.Path,
			name.Line,
			fmt.Sprintf("%s %q is a single letter", name.Kind, name.Name),
			"name it after what it is or holds; a short abbreviation such as srv, row "+
				"or idx is fine, one letter is not",
		))
	}
	return findings
}

func isSingleLetter(name string) bool {
	if utf8.RuneCountInString(name) != 1 {
		return false
	}
	letter, _ := utf8.DecodeRuneInString(name)
	return unicode.IsLetter(letter)
}
