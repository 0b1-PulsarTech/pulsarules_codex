package pproflabels

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

type Analyzer struct {
	reporter core.Reporter
}

func NewAnalyzer() *Analyzer {
	return &Analyzer{
		// SeverityError, unlike every other CatAST reporter: a PII label leaking into
		// a panic traceback is a security incident, not a style nit, so it blocks.
		reporter: core.NewReporter("pprof-labels", core.SeverityError, core.CatAST),
	}
}

func (a *Analyzer) ID() string          { return "pprof-labels" }
func (a *Analyzer) Stage() core.StageID { return core.StageAST }

func (a *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	reporter := a.reporter.Resolved(ctx)
	return core.RunPerGoFile(ctx, func(fc core.FileChange, f *ast.File) []core.Finding {
		return checkFile(ctx.ASTCache.FileSet(), fc, f, reporter)
	})
}

type pprofCheck struct {
	fset      *token.FileSet
	fc        core.FileChange
	pprofName string
	reporter  core.Reporter
}

func checkFile(
	fset *token.FileSet, fc core.FileChange, file *ast.File, reporter core.Reporter,
) []core.Finding {
	pprofName, ok := pprofImportName(file)
	if !ok {
		return nil
	}
	check := pprofCheck{fset: fset, fc: fc, pprofName: pprofName, reporter: reporter}

	var findings []core.Finding
	ast.Inspect(file, func(node ast.Node) bool {
		params, body := funcSignature(node)
		if body == nil {
			return true
		}
		findings = append(findings, checkFuncBody(check, params, body)...)
		return false
	})
	return findings
}

func checkFuncBody(
	check pprofCheck, params *ast.FieldList, body *ast.BlockStmt,
) []core.Finding {
	shadowed := isNameShadowed(params, body, check.pprofName)

	var findings []core.Finding
	ast.Inspect(body, func(node ast.Node) bool {
		if funcLit, ok := node.(*ast.FuncLit); ok {
			findings = append(
				findings, checkFuncBody(check, funcLit.Type.Params, funcLit.Body)...,
			)
			return false
		}
		call, ok := node.(*ast.CallExpr)
		if !ok || shadowed || !isPprofLabelsCall(call, check.pprofName) {
			return true
		}
		for _, key := range piiKeyArgs(call) {
			findings = append(findings, check.reporter.At(
				check.fc.Path,
				check.fset.Position(call.Pos()).Line,
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

func isPprofLabelsCall(call *ast.CallExpr, pprofName string) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil || sel.Sel.Name != "Labels" {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == pprofName
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
