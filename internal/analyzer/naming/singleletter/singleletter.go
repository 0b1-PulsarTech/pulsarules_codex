package singleletter

import (
	"fmt"
	"slices"
	"unicode"
	"unicode/utf8"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

const ID = "single-letter-name"

const (
	paramExportedOnly = "exported_only"
	paramAllowedNames = "allowed_names"
)

type Analyzer struct{}

func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

func (analyzer *Analyzer) reporter() finding.Reporter {
	return finding.NewReporter(ID, finding.SeverityWarning, finding.CatAST)
}

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []finding.Finding {
	params := ctx.Params(ID)
	cfg := settings{
		reporter:     analyzer.reporter(),
		exportedOnly: params.Bool(paramExportedOnly, false),
		allowed:      params.Strings(paramAllowedNames, nil),
	}
	return ctx.EachSyntaxFile(func(fc pass.FileChange, file *syntax.File) []finding.Finding {
		if fc.Generated {
			return nil
		}
		return checkFile(fc, file, cfg)
	})
}

type settings struct {
	reporter     finding.Reporter
	exportedOnly bool
	allowed      []string
}

func checkFile(fc pass.FileChange, file *syntax.File, cfg settings) []finding.Finding {
	findings := make([]finding.Finding, 0, len(file.Names))
	for _, name := range file.Names {
		if name.Exempt || !isSingleLetter(name.Name) ||
			(cfg.exportedOnly && !name.Exported) || slices.Contains(cfg.allowed, name.Name) {
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

func Describe() pass.Descriptor {
	return pass.Descriptor{
		ID:     ID,
		Rule:   "naming",
		Params: []string{paramExportedOnly, paramAllowedNames},
	}
}
