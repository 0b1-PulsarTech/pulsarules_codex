package complexity

import (
	"fmt"
	"go/ast"
	"go/token"
	"math"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

// testFuncLinesMultiplier is the allowance a _test.go function gets over
// max_func_lines. why: a table-driven test carries a literal case table with
// no code counterpart, the same reason filesize relaxes its whole-file limit
// for tests; complexity/param-count stay unrelaxed since those measure real
// branching. 1.5x clears this repo's worst pre-fix overage (85/80) with headroom.
const testFuncLinesMultiplier = 1.5

// thresholds bundles the three configurable limits, plus the two reporters
// already resolved against the run's config, so they thread through
// checkFile/checkFuncDecl as one parameter instead of five.
type thresholds struct {
	maxComplexity int
	maxFuncLines  int
	maxParams     int
	warnReporter  core.Reporter
	infoReporter  core.Reporter
}

// maxFuncLinesFor reads as a lookup on the file rather than a boolean handed
// to a function (see static/filesize.lineLimits.forFile for the same shape).
func (th thresholds) maxFuncLinesFor(fc core.FileChange) int {
	if fc.IsTest {
		return int(math.Round(float64(th.maxFuncLines) * testFuncLinesMultiplier))
	}
	return th.maxFuncLines
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
