package complexity

import (
	"fmt"
	"go/ast"
	"go/token"
	"math"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

type thresholds struct {
	maxComplexity int
	maxFuncLines  int
	maxParams     int
	warnReporter  finding.Reporter
	infoReporter  finding.Reporter
}

func (th thresholds) maxFuncLinesFor(fc pass.FileChange) int {
	if fc.IsTest {
		const testFuncLinesMultiplier = 1.5
		// A table-driven test carries a literal case table with no code counterpart;
		// complexity and param-count stay unrelaxed since those measure real branching.
		return int(math.Round(float64(th.maxFuncLines) * testFuncLinesMultiplier))
	}
	return th.maxFuncLines
}

func (th thresholds) checkFile(
	fset *token.FileSet,
	fc pass.FileChange,
	file *ast.File,
) []finding.Finding {
	if fset == nil {
		return nil
	}
	var findings []finding.Finding

	for _, decl := range file.Decls {
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
	fc pass.FileChange,
	fn *ast.FuncDecl,
) []finding.Finding {
	var findings []finding.Finding
	comp := cyclomaticComplexity(fn)
	if comp > th.maxComplexity {
		findings = append(findings, th.warnReporter.At(
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
	maxFuncLines := th.maxFuncLinesFor(fc)
	if funcLines > maxFuncLines {
		findings = append(findings, th.warnReporter.At(
			fc.Path,
			fset.Position(fn.Pos()).Line,
			fmt.Sprintf(
				"%s is %d lines, max %d",
				fn.Name.Name, funcLines, maxFuncLines,
			),
			"extract helper functions to reduce size",
		))
	}

	if fn.Type.Params != nil {
		paramCount := countParams(fn.Type.Params)
		if paramCount > th.maxParams {
			findings = append(findings, th.warnReporter.At(
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

	findings = append(findings, checkFlagArguments(fset, fc, fn, th.warnReporter)...)
	findings = append(findings, findMagicNumbers(fset, fc, fn, th.infoReporter)...)
	return findings
}
