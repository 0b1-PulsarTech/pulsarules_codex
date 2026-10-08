package registry

import (
	"errors"
	"fmt"
	"go/ast"
	"go/build"
	"go/constant"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io/fs"
	"maps"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

const (
	analyzerTree = "../../analyzer"
	analyzerPath = "github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer"
)

func analyzerDirs() map[string][]string {
	gates := []Gate{GateStatic, GateArch, GateFullOnly}
	dirs := map[string][]string{}
	for _, analyzer := range Analyzers(nil, core.NewLanguageRegistry(), nil, gates) {
		pkgPath := reflect.Indirect(reflect.ValueOf(analyzer)).Type().PkgPath()
		dir := strings.TrimPrefix(pkgPath, analyzerPath+"/")
		dirs[dir] = append(dirs[dir], analyzer.ID())
	}
	return dirs
}

type callSite struct {
	call *ast.CallExpr
	fn   *ast.FuncDecl
}

type typedPackage struct {
	dir     string
	files   []*ast.File
	info    *types.Info
	callees map[*ast.Ident]callSite
	decls   map[types.Object]*ast.FuncDecl
}

type fixtureImporter map[string]*types.Package

func (imp fixtureImporter) Import(path string) (*types.Package, error) {
	if pkg, found := imp[path]; found {
		return pkg, nil
	}
	return nil, fmt.Errorf("package %q is not part of the fixture", path)
}

func typeCheck(
	t *testing.T, fset *token.FileSet, dirs map[string][]*ast.File, imp types.Importer,
) []*typedPackage {
	t.Helper()
	packages := make([]*typedPackage, 0, len(dirs))
	for _, dir := range slices.Sorted(maps.Keys(dirs)) {
		pkg := &typedPackage{
			dir: dir, files: dirs[dir], callees: map[*ast.Ident]callSite{},
			decls: map[types.Object]*ast.FuncDecl{}, info: &types.Info{
				Types: map[ast.Expr]types.TypeAndValue{}, Defs: map[*ast.Ident]types.Object{},
				Uses:       map[*ast.Ident]types.Object{},
				Selections: map[*ast.SelectorExpr]*types.Selection{},
			},
		}
		var failure error
		collect := func(err error) { failure = errors.Join(failure, err) }
		conf := types.Config{Importer: imp, Error: collect}
		checked, _ := conf.Check(analyzerPath+"/"+dir, fset, pkg.files, pkg.info)
		if failure != nil {
			t.Fatalf("type-check %s: %v", dir, failure)
		}
		if fixture, isFixture := imp.(fixtureImporter); isFixture {
			fixture[checked.Path()] = checked
		}
		pkg.index()
		packages = append(packages, pkg)
	}
	return packages
}

func (pkg *typedPackage) index() {
	for _, file := range pkg.files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			pkg.decls[pkg.info.Defs[fn.Name]] = fn
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				if call, isCall := node.(*ast.CallExpr); isCall {
					pkg.callees[calleeIdent(call)] = callSite{call: call, fn: fn}
				}
				return true
			})
		}
	}
}

func calleeIdent(call *ast.CallExpr) *ast.Ident {
	fun := ast.Unparen(call.Fun)
	switch index := fun.(type) {
	case *ast.IndexExpr:
		fun = index.X
	case *ast.IndexListExpr:
		fun = index.X
	}
	switch callee := ast.Unparen(fun).(type) {
	case *ast.Ident:
		return callee
	case *ast.SelectorExpr:
		return callee.Sel
	}
	return nil
}

func loadTree(t *testing.T) []*typedPackage {
	t.Helper()
	fset, dirs := token.NewFileSet(), map[string][]*ast.File{}
	walk := func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() || strings.HasSuffix(path, "_test.go") {
			return walkErr
		}
		if match, _ := build.Default.MatchFile(filepath.Dir(path), entry.Name()); !match {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		dir, _ := filepath.Rel(analyzerTree, filepath.Dir(path))
		dirs[filepath.ToSlash(dir)] = append(dirs[filepath.ToSlash(dir)], file)
		return err
	}
	if err := filepath.WalkDir(analyzerTree, walk); err != nil {
		t.Fatalf("parse analyzer tree: %v", err)
	}
	return typeCheck(t, fset, dirs, importer.ForCompiler(fset, "source", nil))
}

type scanResult struct {
	got      map[[2]string]bool
	failures []string
}

func (result *scanResult) add(where string, ids, keys []string, err error) {
	if err != nil {
		result.failures = append(result.failures, where+": "+err.Error())
	}
	for _, id := range ids {
		for _, key := range keys {
			result.got[[2]string{id, key}] = true
		}
	}
}

func coreName(obj types.Object) string {
	const corePath = analyzerPath + "/core"
	fn, ok := obj.(*types.Func)
	if !ok || fn.Pkg() == nil || fn.Pkg().Path() != corePath {
		return ""
	}
	bare := strings.NewReplacer("(", "", ")", "", "*", "").Replace(fn.FullName())
	return strings.TrimPrefix(bare, corePath+".")
}

func isReader(name string) bool {
	readers := []string{"ParamSet.Int", "ParamSet.Bool", "ParamSet.Strings", "ParamSet.String"}
	return name == "ExtensionFilter" || slices.Contains(readers, name)
}

func isParamSetType(typ types.Type) bool {
	named, isNamed := types.Unalias(typ).(*types.Named)
	if !isNamed || named.Obj().Pkg() == nil {
		return false
	}
	return named.Obj().Pkg().Path() == analyzerPath+"/core" && named.Obj().Name() == "ParamSet"
}

func scanParams(packages []*typedPackage, dirIDs map[string][]string) scanResult {
	result := scanResult{got: map[[2]string]bool{}}
	for _, pkg := range packages {
		if pkg.dir != "core" {
			pkg.scanParamUses(&result, dirIDs[pkg.dir])
		}
		for ident, obj := range pkg.info.Uses {
			site, called := pkg.callees[ident]
			name := coreName(obj)
			switch {
			case isReader(name) && !called:
				err := fmt.Errorf("%s is used as a value, not called", ident.Name)
				result.add(pkg.dir, nil, nil, err)
			case isReader(name):
				pkg.scanCall(site, name, dirIDs[pkg.dir], &result)
			case name == "AnalysisContext.Params":
				pkg.scanParamsCall(site, &result)
			}
		}
	}
	return result
}

func (pkg *typedPackage) scanCall(site callSite, name string, dirIDs []string, out *scanResult) {
	call, fn := site.call, site.fn
	where := pkg.dir + " " + fn.Name.Name
	if name == "ExtensionFilter" {
		ids, err := pkg.resolve(call.Args[1])
		out.add(where, ids, []string{"extensions"}, err)
		return
	}
	sel, _ := ast.Unparen(call.Fun).(*ast.SelectorExpr)
	selection := pkg.info.Selections[sel]
	if selection == nil || selection.Kind() != types.MethodVal {
		out.add(where, nil, nil, errors.New("a method expression reads params"))
		return
	}
	if pkg.dir == "core" {
		if !slices.Contains([]string{"ExtensionFilter", "ParamSet.Severity"},
			coreName(pkg.info.Defs[fn.Name])) {
			out.add(where, nil, nil, errors.New("core reads params outside its modeled readers"))
		}
		return
	}
	keys, keyErr := pkg.resolve(call.Args[0])
	ids, idErr := pkg.attribute(sel.X, fn, dirIDs)
	out.add(where, ids, keys, errors.Join(keyErr, idErr))
}

func (pkg *typedPackage) attribute(
	receiver ast.Expr, fn *ast.FuncDecl, dirIDs []string,
) ([]string, error) {
	if call := pkg.paramsCall(receiver, fn); call != nil {
		return pkg.resolve(call.Args[0])
	}
	if len(dirIDs) == 1 {
		return dirIDs, pkg.foreignParams(receiver, fn)
	}
	return nil, fmt.Errorf("a read in %s is not attributable to one of %v", pkg.dir, dirIDs)
}

func (pkg *typedPackage) paramsCall(receiver ast.Expr, fn *ast.FuncDecl) *ast.CallExpr {
	receiver = ast.Unparen(receiver)
	if ident, isIdent := receiver.(*ast.Ident); isIdent {
		receiver = pkg.onlyAssignment(ident, fn)
	}
	call, isCall := receiver.(*ast.CallExpr)
	if !isCall || coreName(pkg.info.Uses[calleeIdent(call)]) != "AnalysisContext.Params" {
		return nil
	}
	return call
}

func boundOutside(target types.Object, ident *ast.Ident, fn *ast.FuncDecl) bool {
	signatures := []ast.Node{fn.Type}
	if fn.Recv != nil {
		signatures = append(signatures, fn.Recv)
	}
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		lit, isLit := node.(*ast.FuncLit)
		if isLit && ident.Pos() >= lit.Pos() && ident.End() <= lit.End() {
			signatures = append(signatures, lit.Type)
		}
		return true
	})
	return target.Parent() == target.Pkg().Scope() ||
		slices.ContainsFunc(signatures, func(sig ast.Node) bool {
			return target.Pos() >= sig.Pos() && target.Pos() < sig.End()
		})
}

func (pkg *typedPackage) onlyAssignment(ident *ast.Ident, fn *ast.FuncDecl) ast.Expr {
	target := pkg.info.Uses[ident]
	if target == nil || boundOutside(target, ident, fn) {
		return nil
	}
	values := pkg.assignedValues(target, fn)
	if len(values) != 1 {
		return nil
	}
	return values[0]
}

func (pkg *typedPackage) assignedValues(target types.Object, fn *ast.FuncDecl) []ast.Expr {
	var values []ast.Expr
	record := func(names []*ast.Ident, rhs []ast.Expr) {
		for idx, name := range names {
			if target != pkg.info.Defs[name] && target != pkg.info.Uses[name] {
				continue
			}
			var value ast.Expr
			if len(rhs) == len(names) {
				value = rhs[idx]
			}
			values = append(values, value)
		}
	}
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		switch stmt := node.(type) {
		case *ast.AssignStmt:
			record(identsOf(stmt.Lhs), stmt.Rhs)
		case *ast.ValueSpec:
			record(stmt.Names, stmt.Values)
		case *ast.RangeStmt:
			record(identsOf([]ast.Expr{stmt.Key, stmt.Value}), nil)
		case *ast.UnaryExpr:
			if stmt.Op == token.AND {
				record(identsOf([]ast.Expr{ast.Unparen(stmt.X)}), nil)
			}
		}
		return true
	})
	return values
}

func identsOf(exprs []ast.Expr) []*ast.Ident {
	names := make([]*ast.Ident, len(exprs))
	for idx, expr := range exprs {
		names[idx], _ = expr.(*ast.Ident)
	}
	return names
}

func (pkg *typedPackage) resolve(expr ast.Expr) ([]string, error) {
	if value := pkg.info.Types[expr].Value; value != nil && value.Kind() == constant.String {
		return []string{constant.StringVal(value)}, nil
	}
	if call, isCall := ast.Unparen(expr).(*ast.CallExpr); isCall {
		decl := pkg.decls[pkg.info.Uses[calleeIdent(call)]]
		if decl != nil && decl.Name.Name == "ID" && len(decl.Body.List) == 1 {
			if ret, ok := decl.Body.List[0].(*ast.ReturnStmt); ok && len(ret.Results) == 1 {
				return pkg.resolve(ret.Results[0])
			}
		}
	}
	return nil, fmt.Errorf("%s is not a constant", types.ExprString(expr))
}

func compareParams(result scanResult, declared map[string][]string) []string {
	want := map[[2]string]bool{}
	for id, keys := range declared {
		for _, key := range keys {
			want[[2]string{id, key}] = true
		}
	}
	problems := slices.Clone(result.failures)
	for pair := range result.got {
		if !want[pair] {
			problems = append(problems, fmt.Sprintf("analyzer %q reads %q, not declared",
				pair[0], pair[1]))
		}
	}
	for pair := range want {
		if !result.got[pair] {
			problems = append(problems, fmt.Sprintf("analyzer %q declares %q, nothing reads it",
				pair[0], pair[1]))
		}
	}
	slices.Sort(problems)
	return problems
}

func TestParamsMatchEachAnalyzerExactly(t *testing.T) {
	t.Parallel()

	result := scanParams(loadTree(t), analyzerDirs())
	if len(result.got) == 0 {
		t.Fatal("scan found no param reads; the scanner is broken")
	}
	for _, problem := range compareParams(result, Params()) {
		t.Error(problem)
	}
}

func TestParamsNameRegisteredAnalyzers(t *testing.T) {
	t.Parallel()

	for id, keys := range Params() {
		if !slices.Contains(IDs(), id) || len(keys) == 0 {
			t.Errorf("Params()[%q] = %v, want keys of a registered analyzer", id, keys)
		}
	}
}

func TestScannedDirsCoverRegistry(t *testing.T) {
	t.Parallel()

	loaded := map[string]bool{}
	for _, pkg := range loadTree(t) {
		loaded[pkg.dir] = true
	}
	var scanned []string
	for dir, ids := range analyzerDirs() {
		scanned = append(scanned, ids...)
		if !loaded[dir] {
			t.Errorf("analyzer dir %q is not in the loaded tree", dir)
		}
	}
	slices.Sort(scanned)
	if want := slices.Sorted(slices.Values(IDs())); !slices.Equal(scanned, want) {
		t.Errorf("scanned analyzer IDs = %v, want every registered ID %v", scanned, want)
	}
}
