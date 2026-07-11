package complexity

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

// thresholds bundles the three configurable limits so they thread through
// checkFile/checkFuncDecl as one parameter instead of three.
type thresholds struct {
	maxComplexity int
	maxFuncLines  int
	maxParams     int
}

func (th thresholds) checkFile(
	fset *token.FileSet,
	fc core.FileChange,
	f *ast.File,
) []core.Finding {
	if fset == nil {
		return nil
	}
	var findings []core.Finding

	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		findings = append(findings, th.checkFuncDecl(fset, fc, fn)...)
	}

	return findings
}

func (th thresholds) checkFuncDecl(
	fset *token.FileSet,
	fc core.FileChange,
	fn *ast.FuncDecl,
) []core.Finding {
	var findings []core.Finding

	comp := cyclomaticComplexity(fn)
	if comp > th.maxComplexity {
		findings = append(findings, complexityWarnReporter.At(
			fc.Path,
			fset.Position(fn.Pos()).Line,
			fmt.Sprintf(
				"%s has cyclomatic complexity %d, max %d",
				fn.Name.Name, comp, th.maxComplexity,
			),
			"extract conditionals into helper functions or simplify logic",
		))
	}

	startLine := fset.Position(fn.Body.Lbrace).Line
	endLine := fset.Position(fn.Body.Rbrace).Line
	funcLines := endLine - startLine + 1
	if funcLines > th.maxFuncLines {
		findings = append(findings, complexityWarnReporter.At(
			fc.Path,
			fset.Position(fn.Pos()).Line,
			fmt.Sprintf(
				"%s is %d lines, max %d",
				fn.Name.Name, funcLines, th.maxFuncLines,
			),
			"extract helper functions to reduce size",
		))
	}

	if fn.Type.Params != nil {
		paramCount := countParams(fn.Type.Params)
		if paramCount > th.maxParams {
			findings = append(findings, complexityWarnReporter.At(
				fc.Path,
				fset.Position(fn.Pos()).Line,
				fmt.Sprintf(
					"%s has %d parameters, max %d",
					fn.Name.Name, paramCount, th.maxParams,
				),
				"use a parameter struct or split the function",
			))
		}
	}

	findings = append(findings, checkFlagArguments(fset, fc, fn)...)
	findings = append(findings, findMagicNumbers(fset, fc, fn)...)

	return findings
}
