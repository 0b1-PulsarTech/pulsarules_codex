package pproflabel

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/goast"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

const ID = "pprof-labels"

type Analyzer struct {
	reporter finding.Reporter
}

func NewAnalyzer() *Analyzer {
	return &Analyzer{
		// SeverityError, unlike every other CatAST reporter: a PII label leaking into
		// a panic traceback is a security incident, not a style nit, so it blocks.
		reporter: finding.NewReporter(ID, finding.SeverityError, finding.CatAST),
	}
}

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []finding.Finding {
	reporter := pass.ResolveReporter(analyzer.reporter, ctx)
	return pass.RunPerGoFile(ctx, func(fc pass.FileChange, file *ast.File) []finding.Finding {
		return checkFile(ctx.ASTCache.FileSet(), fc, file, reporter)
	})
}

func checkFile(
	fset *token.FileSet, fc pass.FileChange, file *ast.File, reporter finding.Reporter,
) []finding.Finding {
	var findings []finding.Finding
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || !goast.PkgCall(file, call, "runtime/pprof", "Labels") {
			return true
		}
		for _, key := range piiKeyArgs(call) {
			findings = append(findings, reporter.At(
				fc.Path,
				fset.Position(call.Pos()).Line,
				fmt.Sprintf(
					"pprof.Labels key %q looks like PII; its value reaches panic "+
						"tracebacks under GOTRACEBACK=all - use an opaque correlation "+
						"id (trace/span id) instead",
					key,
				),
				"replace the label value with an opaque trace/span id",
			))
		}
		return true
	})
	return findings
}

func piiKeyArgs(call *ast.CallExpr) []string {
	var matches []string
	for index := 0; index < len(call.Args); index += 2 {
		lit, ok := call.Args[index].(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			continue
		}
		raw, err := strconv.Unquote(lit.Value)
		if err != nil {
			continue
		}
		if isPIILabelKey(normalizeLabelKey(raw)) {
			matches = append(matches, raw)
		}
	}
	return matches
}

func normalizeLabelKey(raw string) string {
	raw = strings.ToLower(raw)
	raw = strings.ReplaceAll(raw, "_", "")
	raw = strings.ReplaceAll(raw, "-", "")
	return raw
}

func isPIILabelKey(key string) bool {
	// PII and secret key names (security.md #9), matched after normalizeLabelKey.
	switch key {
	case "userid", "user", "email", "phone", "token", "password", "passwd", "secret",
		"apikey", "authorization", "ssn", "cpf", "cnpj", "creditcard", "cardnumber":
		return true
	}
	return false
}

func Describe() pass.Descriptor {
	return pass.Descriptor{ID: ID, Rule: "security"}
}
