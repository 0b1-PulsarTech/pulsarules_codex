package packagevar

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

type pkgKey struct {
	dir string
	pkg string
}

type dirInfo struct {
	types    map[string]bool
	errFuncs map[string]bool
}

type dirScan struct {
	dirs map[pkgKey]dirInfo
}

func scanDirs(ctx *pass.Context) dirScan {
	scan := dirScan{dirs: make(map[pkgKey]dirInfo)}
	dirs := dirsWithVars(ctx)
	if len(dirs) == 0 || ctx.Sources == nil {
		return scan
	}
	fset := token.NewFileSet()
	ctx.Sources.Walk(func(file string, _ string) bool {
		file = filepath.ToSlash(file)
		dir := path.Dir(file)
		if !dirs[dir] || path.Ext(file) != ".go" {
			return true
		}
		src, ok := ctx.Sources.Read(file)
		wantDecl := bytes.Contains(src, []byte("type ")) || bytes.Contains(src, []byte("func "))
		if !ok || !wantDecl {
			return true
		}
		parsed, err := parser.ParseFile(fset, file, src, parser.SkipObjectResolution)
		if err != nil {
			return true
		}
		scan.collectDecls(dir, parsed)
		return true
	})
	return scan
}

func (scan dirScan) collectDecls(dir string, file *ast.File) {
	key := pkgKey{dir: dir, pkg: file.Name.Name}
	names, seen := scan.dirs[key]
	if !seen {
		names = dirInfo{types: make(map[string]bool), errFuncs: make(map[string]bool)}
		scan.dirs[key] = names
	}
	for _, decl := range file.Decls {
		switch typed := decl.(type) {
		case *ast.GenDecl:
			for _, spec := range typed.Specs {
				if ts, isType := spec.(*ast.TypeSpec); isType {
					names.types[ts.Name.Name] = true
				}
			}
		case *ast.FuncDecl:
			if isErrReturning(typed) {
				names.errFuncs[typed.Name.Name] = true
			}
		}
	}
}

func dirsWithVars(ctx *pass.Context) map[string]bool {
	dirs := make(map[string]bool)
	for fc, file := range ctx.ChangedGoASTs() {
		for _, decl := range file.Decls {
			if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.VAR {
				dirs[dirOf(fc.Path)] = true
			}
		}
	}
	return dirs
}
