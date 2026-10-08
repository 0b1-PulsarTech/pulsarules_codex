package complexity

import (
	"fmt"
	"go/ast"
	"go/token"
	"math"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
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

func (th thresholds) checkFile(fc pass.FileChange, file *syntax.File) []finding.Finding {
	findings := make([]finding.Finding, 0, len(file.Functions))
	for _, fn := range file.Functions {
		findings = append(findings, th.checkTopLevel(fc, fn)...)
	}
	return findings
}

func (th thresholds) checkTopLevel(fc pass.FileChange, fn syntax.Function) []finding.Finding {
	if fn.Nested {
		return nil
	}
	return th.checkFunction(fc, fn)
}

func (th thresholds) checkFunction(fc pass.FileChange, fn syntax.Function) []finding.Finding {
	var findings []finding.Finding
	if fn.Complexity > th.maxComplexity {
		findings = append(findings, th.warnReporter.At(
			fc.Path,
			fn.Line,
			fmt.Sprintf(
				"%s has cyclomatic complexity %d, max %d",
				fn.Name, fn.Complexity, th.maxComplexity,
			),
			"extract conditionals into helper functions or simplify logic",
		))
	}

	funcLines := fn.BodyEnd - fn.BodyStart + 1
	maxFuncLines := th.maxFuncLinesFor(fc)
	// A test container wraps a whole file's cases, so its span says nothing about size.
	if funcLines > maxFuncLines && !fn.TestContainer {
		findings = append(findings, th.warnReporter.At(
			fc.Path,
			fn.Line,
			fmt.Sprintf("%s is %d lines, max %d", fn.Name, funcLines, maxFuncLines),
			"extract helper functions to reduce size",
		))
	}

	if fn.Params > th.maxParams {
		findings = append(findings, th.warnReporter.At(
			fc.Path,
			fn.Line,
			fmt.Sprintf("%s has %d parameters, max %d", fn.Name, fn.Params, th.maxParams),
			"use a parameter struct or split the function",
		))
	}
	return findings
}

func (th thresholds) checkGoFile(
	fset *token.FileSet,
	fc pass.FileChange,
	file *ast.File,
) []finding.Finding {
	decls := bodyDecls(file)
	findings := make([]finding.Finding, 0, len(decls))
	for _, fn := range decls {
		findings = append(findings, th.checkGoFunc(fset, fc, file, fn)...)
	}
	return findings
}

func (th thresholds) checkGoFunc(
	fset *token.FileSet,
	fc pass.FileChange,
	file *ast.File,
	fn *ast.FuncDecl,
) []finding.Finding {
	if fset == nil {
		return nil
	}
	findings := checkFlagArguments(fset, fc, fn, th.warnReporter)
	return append(findings, findMagicNumbers(fset, fc, file, fn, th.infoReporter)...)
}

func bodyDecls(file *ast.File) []*ast.FuncDecl {
	var decls []*ast.FuncDecl
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Body != nil {
			decls = append(decls, fn)
		}
	}
	return decls
}
