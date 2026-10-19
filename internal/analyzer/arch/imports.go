package arch

import (
	"errors"
	"go/ast"
	"go/scanner"
	"go/token"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/gomod"
)

type edge struct {
	from string
	to   string
}

type importSite struct {
	file string
	line int
}

type importGraph struct {
	deps  depGraph
	sites map[edge]importSite
}

func loadImportGraph(index *gomod.Index, modulePath string) importGraph {
	graph := importGraph{deps: make(depGraph), sites: make(map[edge]importSite)}
	fset := index.FileSet()
	for _, dir := range index.Dirs() {
		pkg := packageImport(modulePath, dir)
		for _, file := range index.Files(dir) {
			if strings.HasSuffix(file.Path, "_test.go") {
				continue
			}
			if _, seen := graph.deps[pkg]; !seen {
				graph.deps[pkg] = nil
			}
			if brokenImportBlock(fset, file) {
				continue
			}
			specs, _, _ := importHeader(file.Syntax)
			for _, spec := range specs {
				dep := strings.Trim(spec.Path.Value, `"`)
				if !isLocal(modulePath, dep) || slices.Contains(graph.deps[pkg], dep) {
					continue
				}
				graph.deps[pkg] = append(graph.deps[pkg], dep)
				line := fset.Position(spec.Path.Pos()).Line
				graph.sites[edge{from: pkg, to: dep}] = importSite{file: file.Path, line: line}
			}
		}
	}
	return graph
}

func brokenImportBlock(fset *token.FileSet, file gomod.File) bool {
	var errs scanner.ErrorList
	if !errors.As(file.Err, &errs) {
		return false
	}
	// Like go list: only an error before the end of the import block drops the imports.
	// An error at the end of a file with no body decl is still inside the import block.
	_, end, endOfFile := importHeader(file.Syntax)
	limit := fset.PositionFor(end, false).Offset
	return slices.ContainsFunc(errs, func(found *scanner.Error) bool {
		return found.Pos.Offset < limit || (endOfFile && found.Pos.Offset == limit)
	})
}

func importHeader(syntax *ast.File) (specs []*ast.ImportSpec, end token.Pos, endOfFile bool) {
	end, endOfFile = syntax.FileEnd, true
	for _, decl := range syntax.Decls {
		// Like go/build's header reader: an import after another declaration is not one.
		gen, isGen := decl.(*ast.GenDecl)
		if !isGen || gen.Tok != token.IMPORT {
			return specs, decl.Pos(), false
		}
		for _, spec := range gen.Specs {
			if imported, isImport := spec.(*ast.ImportSpec); isImport {
				specs = append(specs, imported)
			}
		}
	}
	return specs, end, endOfFile
}

func packageImport(modulePath, dir string) string {
	if dir == "." {
		return modulePath
	}
	return modulePath + "/" + dir
}

func isLocal(modulePath, importPath string) bool {
	return importPath == modulePath || strings.HasPrefix(importPath, modulePath+"/")
}
