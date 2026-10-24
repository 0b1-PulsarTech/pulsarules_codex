package complexity

import (
	"go/ast"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

const MagicID = "magic-number"

type MagicNumberAnalyzer struct {
	reporter finding.Reporter
}

func NewMagicNumberAnalyzer() *MagicNumberAnalyzer {
	return &MagicNumberAnalyzer{
		reporter: finding.NewReporter(MagicID, finding.SeverityInfo, finding.CatAST),
	}
}

func (analyzer *MagicNumberAnalyzer) Analyze(ctx *pass.Context) []finding.Finding {
	if ctx.ASTCache == nil {
		return nil
	}
	fset := ctx.ASTCache.FileSet()
	return ctx.EachGoFile(func(fc pass.FileChange, file *ast.File) []finding.Finding {
		decls := bodyDecls(file)
		findings := make([]finding.Finding, 0, len(decls))
		for _, fn := range decls {
			findings = append(findings, findMagicNumbers(fset, fc, file, fn, analyzer.reporter)...)
		}
		return findings
	})
}

func DescribeMagicNumber() pass.Descriptor {
	return pass.Descriptor{
		ID:     MagicID,
		Rule:   "code-smells",
		Params: []string{"allowed_numbers"},
	}
}
