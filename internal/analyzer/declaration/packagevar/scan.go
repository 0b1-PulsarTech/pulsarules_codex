package packagevar

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

type fileScan struct {
	fset      *token.FileSet
	path      string
	file      *ast.File
	reporter  pass.Reporter
	reported  map[site]bool
	assigned  map[string]bool
	imports   map[string]bool
	types     map[string]bool
	errFuncs  map[string]bool
	errorsPkg string
	fmtPkg    string
	embeds    []*ast.CommentGroup
	enums     map[string]bool
}

func (scan fileScan) run() []pass.Finding {
	var findings []pass.Finding
	prevEnd := token.NoPos
	for _, decl := range scan.file.Decls {
		if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.VAR {
			findings = append(findings, scan.genFindings(gen, prevEnd)...)
		}
		prevEnd = decl.End()
	}
	return findings
}

func (scan fileScan) genFindings(gen *ast.GenDecl, declPrevEnd token.Pos) []pass.Finding {
	var findings []pass.Finding
	prevEnd, specPos := declPrevEnd, gen.Pos()
	if gen.Lparen.IsValid() {
		prevEnd = gen.Lparen
	}
	for _, spec := range gen.Specs {
		vs, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		if gen.Lparen.IsValid() {
			specPos = vs.Pos()
		}
		if !scan.hasEmbed(prevEnd, specPos) {
			findings = append(findings, scan.nameFindings(vs)...)
		}
		prevEnd = vs.End()
	}
	return findings
}

func (scan fileScan) nameFindings(vs *ast.ValueSpec) []pass.Finding {
	var findings []pass.Finding
	blankReported := false
	for idx, name := range vs.Names {
		if scan.isSpecExempt(vs, idx) || scan.isNameExempt(vs, idx) {
			continue
		}
		isBlank := name.Name == "_"
		if isBlank && blankReported {
			continue
		}
		blankReported = blankReported || isBlank
		line := scan.fset.Position(name.Pos()).Line
		if scan.reported[site{file: scan.path, line: line, name: name.Name}] {
			continue
		}
		suggestion := "move it into the function or struct that owns it, or make it a const"
		if isBlank {
			suggestion = blankSuggestion(vs)
		}
		finding := scan.reporter.At(scan.path, line, "package-level var "+name.Name, suggestion)
		finding.Symbol = name.Name
		findings = append(findings, finding)
	}
	return findings
}

func blankSuggestion(vs *ast.ValueSpec) string {
	const generic = "call it from a constructor or main instead of an init side effect"
	if len(vs.Values) != 1 {
		return generic
	}
	call, ok := unparen(vs.Values[0]).(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return generic
	}
	if _, isSel := unparen(call.Fun).(*ast.SelectorExpr); !isSel {
		return generic
	}
	return generic + "; to assert an interface write `var _ " +
		types.ExprString(call.Fun) + " = " + types.ExprString(call.Args[0]) + "`"
}

func (scan fileScan) isNameExempt(vs *ast.ValueSpec, idx int) bool {
	name := vs.Names[idx].Name
	switch {
	case name != "_":
		return scan.assigned[name] || scan.enums[name]
	case len(vs.Values) == len(vs.Names):
		return scan.isConversionOrComposite(vs.Values[idx])
	}
	return scan.isAssertion(vs) || hasNamedSibling(vs)
}

func hasNamedSibling(vs *ast.ValueSpec) bool {
	for _, name := range vs.Names {
		if name.Name != "_" {
			return true
		}
	}
	return false
}
