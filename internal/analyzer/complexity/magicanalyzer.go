package complexity

import (
	"fmt"
	"go/ast"
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
)

const MagicID = "magic-number"

const paramAllowedNumbers = "allowed_numbers"

type MagicNumberAnalyzer struct {
	reporter finding.Reporter
}

func NewMagicNumberAnalyzer() *MagicNumberAnalyzer {
	return &MagicNumberAnalyzer{
		reporter: finding.NewReporter(MagicID, finding.SeverityInfo, finding.CatAST),
	}
}

func (analyzer *MagicNumberAnalyzer) Analyze(ctx *pass.Context) []finding.Finding {
	allowed := ctx.Params(MagicID).Strings(paramAllowedNumbers, defaultAllowedNumbers())
	findings := analyzer.analyzeSyntax(ctx, allowed)
	if ctx.ASTCache == nil {
		return findings
	}
	fset := ctx.ASTCache.FileSet()
	return append(
		findings,
		ctx.EachGoFile(func(fc pass.FileChange, file *ast.File) []finding.Finding {
			decls := bodyDecls(file)
			goFindings := make([]finding.Finding, 0, len(decls))
			for _, fn := range decls {
				goFindings = append(
					goFindings,
					findMagicNumbers(
						fset,
						fc,
						file,
						fn,
						magicSettings{
							reporter: analyzer.reporter,
							allowed:  allowed,
						},
					)...,
				)
			}
			return goFindings
		})...)
}

func (analyzer *MagicNumberAnalyzer) analyzeSyntax(
	ctx *pass.Context, allowed []string,
) []finding.Finding {
	return ctx.EachSyntaxFile(func(fc pass.FileChange, file *syntax.File) []finding.Finding {
		// Go literals come from the AST walk, which knows const blocks and strconv.
		if fc.IsTest || file.Language == language.Go {
			return nil
		}
		var findings []finding.Finding
		for _, number := range file.Numbers {
			if slices.Contains(allowed, number.Value) {
				continue
			}
			findings = append(findings, analyzer.reporter.At(
				fc.Path,
				number.Line,
				fmt.Sprintf("magic number %s", number.Value),
				"name it with a const declaration",
			))
		}
		return findings
	})
}

func DescribeMagicNumber() pass.Descriptor {
	return pass.Descriptor{
		ID:     MagicID,
		Rule:   "code-smells",
		Params: []string{paramAllowedNumbers},
	}
}
