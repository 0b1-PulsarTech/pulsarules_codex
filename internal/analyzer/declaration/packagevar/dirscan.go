package packagevar

import (
	"go/ast"
	"go/token"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

type pkgKey struct {
	dir string
	pkg string
}

type dirInfo struct {
	types    map[string]bool
	errFuncs map[string]bool
	consts   map[string]bool
}

type dirScan struct {
	dirs     map[pkgKey]dirInfo
	assigned map[pkgKey]map[string]bool
}

func scanDirs(ctx *pass.Context) dirScan {
	scan := dirScan{
		dirs:     make(map[pkgKey]dirInfo),
		assigned: make(map[pkgKey]map[string]bool),
	}
	dirs := dirsWithVars(ctx)
	if len(dirs) == 0 || ctx.Sources == nil {
		return scan
	}
	index := ctx.GoPackages()
	for dir := range dirs {
		for _, file := range index.Files(dir) {
			if file.Err != nil {
				continue
			}
			scan.collectDecls(dir, file.Syntax)
			if isTestFile(file.Path) {
				scan.collectMain(dir, file.Syntax)
			}
		}
	}
	return scan
}

func (scan dirScan) collectDecls(dir string, file *ast.File) {
	key := pkgKey{dir: dir, pkg: file.Name.Name}
	names, seen := scan.dirs[key]
	if !seen {
		names = dirInfo{
			types:    make(map[string]bool),
			errFuncs: make(map[string]bool),
			consts:   make(map[string]bool),
		}
		scan.dirs[key] = names
	}
	for _, decl := range file.Decls {
		switch typed := decl.(type) {
		case *ast.GenDecl:
			collectGen(names, typed)
		case *ast.FuncDecl:
			if isErrReturning(typed) {
				names.errFuncs[typed.Name.Name] = true
			}
		}
	}
}

func collectGen(names dirInfo, gen *ast.GenDecl) {
	for _, spec := range gen.Specs {
		switch typed := spec.(type) {
		case *ast.TypeSpec:
			names.types[typed.Name.Name] = true
		case *ast.ValueSpec:
			if gen.Tok != token.CONST {
				continue
			}
			for _, name := range typed.Names {
				names.consts[name.Name] = true
			}
		}
	}
}

func (scan dirScan) collectMain(dir string, file *ast.File) {
	key := pkgKey{dir: dir, pkg: file.Name.Name}
	if scan.assigned[key] == nil {
		scan.assigned[key] = make(map[string]bool)
	}
	for _, decl := range file.Decls {
		if fn, isFunc := decl.(*ast.FuncDecl); isFunc && isTestMain(fn) {
			collectAssigned(fn.Body, scan.assigned[key])
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
