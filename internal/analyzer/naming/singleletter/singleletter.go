package singleletter

import (
	"fmt"
	"unicode"
	"unicode/utf8"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/syntaxfiles"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

type Analyzer struct{}

func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

func (analyzer *Analyzer) ID() string          { return "single-letter-name" }
func (analyzer *Analyzer) Stage() pass.StageID { return pass.StageAST }

func (analyzer *Analyzer) reporter() pass.Reporter {
	return pass.NewReporter(analyzer.ID(), pass.SeverityWarning, pass.CatAST)
}

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []pass.Finding {
	cfg := settings{
		reporter:     pass.ResolveReporter(analyzer.reporter(), ctx),
		exportedOnly: ctx.Params(analyzer.ID()).Bool("exported_only", false),
	}
	return syntaxfiles.Each(ctx, func(fc pass.FileChange, file *syntax.File) []pass.Finding {
		if fc.Generated {
			return nil
		}
		return checkFile(fc, file, cfg)
	})
}

type settings struct {
	reporter     pass.Reporter
	exportedOnly bool
}

func checkFile(fc pass.FileChange, file *syntax.File, cfg settings) []pass.Finding {
	findings := make([]pass.Finding, 0, len(file.Names))
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
