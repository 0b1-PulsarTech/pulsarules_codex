package gosrc

import (
	"go/ast"
	"go/token"
	"go/types"
	"path"
	"slices"
	"strconv"
)

type CaptureSets struct {
	Left  map[string]bool
	Taken map[string]bool
}

func (sets CaptureSets) captures(name string) bool {
	return sets.Taken[name] || (sets.Left[name] && types.Universe.Lookup(name) != nil)
}

func capturesBare(newFile File, newPos int, bind Binding) bool {
	tok := newFile.body[newPos]
	if tok.kind != token.IDENT || (!newFile.unresolved[tok.off] && !newFile.keys[tok.off]) {
		return false
	}
	return bind.Captures.captures(tok.lit)
}

func RelocationUnsafe(oldFile, newFile File) bool {
	for _, spec := range slices.Concat(oldFile.Imports, newFile.Imports) {
		if spec.Name == "." || spec.Path == "C" {
			return true
		}
	}
	for name := range newFile.TopLevel {
		if types.Universe.Lookup(name) != nil {
			return true
		}
	}
	return false
}

func loadsAtInit(file *ast.File, top map[string]bool) bool {
	env := loadEnv{top: top, errorsPkg: errorsName(file), imports: importNames(file)}
	return top["init"] || hasBlankImport(file) ||
		slices.ContainsFunc(varSpecs(file), func(spec *ast.ValueSpec) bool {
			return callsAtLoad(spec.Values, env)
		})
}

func hasBlankImport(file *ast.File) bool {
	return slices.ContainsFunc(file.Imports, func(spec *ast.ImportSpec) bool {
		// The embed package has no init, so importing it only enables //go:embed.
		return spec.Name != nil && spec.Name.Name == "_" && spec.Path.Value != `"embed"`
	})
}

func varSpecs(file *ast.File) []*ast.ValueSpec {
	var specs []*ast.ValueSpec
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			continue
		}
		for _, spec := range gen.Specs {
			if value, isValue := spec.(*ast.ValueSpec); isValue {
				specs = append(specs, value)
			}
		}
	}
	return specs
}

func callsAtLoad(values []ast.Expr, env loadEnv) bool {
	return !inertAll(values, env)
}

func inertAll(exprs []ast.Expr, env loadEnv) bool {
	return !slices.ContainsFunc(exprs, func(expr ast.Expr) bool { return !inertValue(expr, env) })
}

func inertValue(expr ast.Expr, env loadEnv) bool {
	switch typed := expr.(type) {
	case *ast.BasicLit, *ast.Ident, *ast.FuncLit, *ast.ArrayType, *ast.MapType,
		*ast.FuncType, *ast.InterfaceType, *ast.ChanType, *ast.StructType:
		return true
	case *ast.ParenExpr:
		return inertValue(typed.X, env)
	case *ast.SelectorExpr:
		pkg, isIdent := typed.X.(*ast.Ident)
		return isIdent && env.imports[pkg.Name] && !env.top[pkg.Name]
	}
	return inertCompound(expr, env)
}

func inertCompound(expr ast.Expr, env loadEnv) bool {
	switch typed := expr.(type) {
	case *ast.UnaryExpr:
		return typed.Op != token.ARROW && inertValue(typed.X, env)
	case *ast.BinaryExpr:
		return inertBinary(typed, env)
	case *ast.KeyValueExpr:
		return inertValue(typed.Key, env) && inertValue(typed.Value, env)
	case *ast.CompositeLit:
		return inertAll(typed.Elts, env)
	case *ast.CallExpr:
		return isInert(typed.Fun, env) && inertAll(typed.Args, env) && !loadConversion(typed, env)
	}
	return false
}

func inertBinary(expr *ast.BinaryExpr, env loadEnv) bool {
	_, literalRight := expr.Y.(*ast.BasicLit)
	// A zero divisor or negative shift count panics unless the operand is a literal.
	panics := slices.Contains([]token.Token{token.QUO, token.REM, token.SHL, token.SHR}, expr.Op)
	return (literalRight || !panics) && inertValue(expr.X, env) && inertValue(expr.Y, env)
}

func loadConversion(call *ast.CallExpr, env loadEnv) bool {
	switch fun := ast.Unparen(call.Fun).(type) {
	case *ast.ArrayType:
		return fun.Len != nil
	case *ast.StarExpr:
		if len(call.Args) != 1 {
			return true
		}
		nilArg, isIdent := call.Args[0].(*ast.Ident)
		return !isIdent || nilArg.Name != "nil" || env.top["nil"]
	}
	return false
}

type loadEnv struct {
	top       map[string]bool
	errorsPkg string
	imports   map[string]bool
}

func importNames(file *ast.File) map[string]bool {
	names := map[string]bool{}
	for _, spec := range file.Imports {
		unquoted, _ := strconv.Unquote(spec.Path.Value)
		if spec.Name == nil {
			names[path.Base(unquoted)] = true
		} else {
			names[spec.Name.Name] = true
		}
	}
	return names
}

func errorsName(file *ast.File) string {
	for _, spec := range file.Imports {
		if spec.Path.Value != `"errors"` {
			continue
		}
		if spec.Name == nil {
			return "errors"
		}
		return spec.Name.Name
	}
	return ""
}

func isInert(fun ast.Expr, env loadEnv) bool {
	switch typed := ast.Unparen(fun).(type) {
	case *ast.StarExpr, *ast.ArrayType, *ast.MapType, *ast.FuncType,
		*ast.InterfaceType, *ast.ChanType, *ast.StructType:
		return true
	case *ast.Ident:
		_, isType := types.Universe.Lookup(typed.Name).(*types.TypeName)
		return isType && !env.top[typed.Name]
	case *ast.SelectorExpr:
		pkg, isIdent := typed.X.(*ast.Ident)
		return isIdent && pkg.Name == env.errorsPkg &&
			typed.Sel.Name == "New"
	}
	return false
}
