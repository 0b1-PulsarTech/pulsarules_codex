package packagelevelvar

import (
	"go/ast"
	"go/token"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

type fileScan struct {
	fset     *token.FileSet
	path     string
	file     *ast.File
	reporter core.Reporter
	reported map[site]bool
}

func (scan fileScan) run() []core.Finding {
	var findings []core.Finding
	for _, decl := range scan.file.Decls {
		if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.VAR {
			findings = append(findings, scan.genFindings(gen)...)
		}
	}
	return findings
}

func (scan fileScan) genFindings(gen *ast.GenDecl) []core.Finding {
	var findings []core.Finding
	for _, spec := range gen.Specs {
		if vs, ok := spec.(*ast.ValueSpec); ok {
			findings = append(findings, scan.nameFindings(vs)...)
		}
	}
	return findings
}

func (scan fileScan) nameFindings(vs *ast.ValueSpec) []core.Finding {
	var findings []core.Finding
	blankReported := false
	for idx, name := range vs.Names {
		if scan.isNameExempt(vs, idx) {
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
			suggestion = "call it from a constructor or main instead of an init side effect"
		}
		finding := scan.reporter.At(scan.path, line, "package-level var "+name.Name, suggestion)
		finding.Symbol = name.Name
		findings = append(findings, finding)
	}
	return findings
}

func (scan fileScan) isNameExempt(vs *ast.ValueSpec, idx int) bool {
	if vs.Names[idx].Name != "_" || len(vs.Values) == len(vs.Names) {
		return false
	}
	return hasNamedSibling(vs)
}

func hasNamedSibling(vs *ast.ValueSpec) bool {
	for _, name := range vs.Names {
		if name.Name != "_" {
			return true
		}
	}
	return false
}
