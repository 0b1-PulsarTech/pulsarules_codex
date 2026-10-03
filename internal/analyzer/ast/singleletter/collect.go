package singleletter

import (
	"go/ast"
	"go/token"
	"unicode"
	"unicode/utf8"
)

type site struct {
	ident *ast.Ident
	kind  kind
}

type collector struct {
	seen         map[token.Pos]bool
	sites        []site
	testPkgs     map[string]bool
	exportedOnly bool
}

func newCollector(testPkgs map[string]bool) *collector {
	return &collector{seen: map[token.Pos]bool{}, testPkgs: testPkgs}
}

func (scan *collector) add(ident *ast.Ident, siteKind kind) {
	if ident == nil || !isSingleLetter(ident.Name) {
		return
	}
	if scan.seen[ident.Pos()] {
		return
	}
	// Marked before the skip so a later visit under another kind cannot re-add it.
	scan.seen[ident.Pos()] = true
	if scan.skips(ident.Name, siteKind) {
		return
	}
	scan.sites = append(scan.sites, site{ident: ident, kind: siteKind})
}

func (scan *collector) skips(name string, siteKind kind) bool {
	letter, _ := utf8.DecodeRuneInString(name)
	return scan.exportedOnly && (!unicode.IsUpper(letter) || !siteKind.canBeExported())
}

func isSingleLetter(name string) bool {
	if utf8.RuneCountInString(name) != 1 {
		return false
	}
	letter, _ := utf8.DecodeRuneInString(name)
	return unicode.IsLetter(letter)
}

func (scan *collector) addFields(list *ast.FieldList, siteKind kind) {
	if list == nil {
		return
	}
	for _, field := range list.List {
		if siteKind == kindParameter && isTestingType(field.Type, scan.testPkgs) {
			continue
		}
		for _, ident := range field.Names {
			scan.add(ident, siteKind)
		}
	}
}

func (scan *collector) visit(node ast.Node) bool {
	scan.declSite(node)
	scan.stmtSite(node)
	return true
}

func (scan *collector) declSite(node ast.Node) {
	switch decl := node.(type) {
	case *ast.FuncDecl:
		scan.funcDecl(decl)
	case *ast.FuncType:
		scan.addFields(decl.TypeParams, kindTypeParam)
		scan.addFields(decl.Params, kindParameter)
		scan.addFields(decl.Results, kindResult)
	case *ast.TypeSpec:
		scan.add(decl.Name, kindType)
		scan.addFields(decl.TypeParams, kindTypeParam)
	case *ast.GenDecl:
		scan.genDecl(decl)
	case *ast.StructType:
		scan.addFields(decl.Fields, kindField)
	case *ast.InterfaceType:
		scan.addFields(decl.Methods, kindMethod)
	}
}

func (scan *collector) funcDecl(decl *ast.FuncDecl) {
	declKind := kindFunction
	if decl.Recv != nil {
		declKind = kindMethod
	}
	scan.add(decl.Name, declKind)
	scan.addFields(decl.Recv, kindReceiver)
	scan.recvTypeParams(decl.Recv)
}

func (scan *collector) recvTypeParams(recv *ast.FieldList) {
	if recv == nil || len(recv.List) == 0 {
		return
	}
	expr := recv.List[0].Type
	for {
		switch typed := expr.(type) {
		case *ast.StarExpr:
			expr = typed.X
			continue
		case *ast.ParenExpr:
			expr = typed.X
			continue
		case *ast.IndexExpr:
			scan.addExpr(typed.Index, kindTypeParam)
		case *ast.IndexListExpr:
			for _, index := range typed.Indices {
				scan.addExpr(index, kindTypeParam)
			}
		}
		return
	}
}

func (scan *collector) genDecl(decl *ast.GenDecl) {
	declKind := kindVariable
	if decl.Tok == token.CONST {
		declKind = kindConstant
	}
	for _, spec := range decl.Specs {
		switch typed := spec.(type) {
		case *ast.ValueSpec:
			for _, ident := range typed.Names {
				scan.add(ident, declKind)
			}
		case *ast.ImportSpec:
			scan.add(typed.Name, kindImportAlias)
		}
	}
}

func (scan *collector) stmtSite(node ast.Node) {
	switch stmt := node.(type) {
	case *ast.AssignStmt:
		scan.define(stmt, kindVariable)
	case *ast.RangeStmt:
		if stmt.Tok == token.DEFINE {
			scan.addExpr(stmt.Key, kindLoopVariable)
			scan.addExpr(stmt.Value, kindLoopVariable)
		}
	case *ast.ForStmt:
		if initStmt, ok := stmt.Init.(*ast.AssignStmt); ok {
			scan.define(initStmt, kindLoopVariable)
		}
	case *ast.TypeSwitchStmt:
		if assign, ok := stmt.Assign.(*ast.AssignStmt); ok {
			scan.define(assign, kindSwitchBinding)
		}
	case *ast.LabeledStmt:
		scan.add(stmt.Label, kindLabel)
	}
}

func (scan *collector) define(assign *ast.AssignStmt, siteKind kind) {
	if assign.Tok != token.DEFINE {
		return
	}
	for _, lhs := range assign.Lhs {
		scan.addExpr(lhs, siteKind)
	}
}

func (scan *collector) addExpr(expr ast.Expr, siteKind kind) {
	if ident, ok := expr.(*ast.Ident); ok {
		scan.add(ident, siteKind)
	}
}
