package names

import (
	"go/ast"
	"go/token"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

func (scan *collector) declSite(node ast.Node) {
	switch decl := node.(type) {
	case *ast.FuncDecl:
		scan.funcDecl(decl)
	case *ast.FuncType:
		scan.addFields(decl.TypeParams, syntax.NameTypeParam)
		scan.addFields(decl.Params, syntax.NameParameter)
		scan.addFields(decl.Results, syntax.NameResult)
	case *ast.TypeSpec:
		scan.add(decl.Name, syntax.NameType)
		scan.addFields(decl.TypeParams, syntax.NameTypeParam)
	case *ast.GenDecl:
		scan.genDecl(decl)
	case *ast.StructType:
		scan.addFields(decl.Fields, syntax.NameField)
	case *ast.InterfaceType:
		scan.addFields(decl.Methods, syntax.NameMethod)
	}
}

func (scan *collector) funcDecl(decl *ast.FuncDecl) {
	kind := syntax.NameFunction
	if decl.Recv != nil {
		kind = syntax.NameMethod
	}
	scan.add(decl.Name, kind)
	scan.addFields(decl.Recv, syntax.NameReceiver)
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
			scan.addExpr(typed.Index, syntax.NameTypeParam)
		case *ast.IndexListExpr:
			for _, index := range typed.Indices {
				scan.addExpr(index, syntax.NameTypeParam)
			}
		}
		return
	}
}

func (scan *collector) genDecl(decl *ast.GenDecl) {
	kind := syntax.NameVariable
	if decl.Tok == token.CONST {
		kind = syntax.NameConstant
	}
	for _, spec := range decl.Specs {
		switch typed := spec.(type) {
		case *ast.ValueSpec:
			for _, ident := range typed.Names {
				scan.add(ident, kind)
			}
		case *ast.ImportSpec:
			scan.add(typed.Name, syntax.NameImportAlias)
		}
	}
}

func (scan *collector) stmtSite(node ast.Node) {
	switch stmt := node.(type) {
	case *ast.AssignStmt:
		scan.define(stmt, syntax.NameVariable)
	case *ast.RangeStmt:
		if stmt.Tok == token.DEFINE {
			scan.addExpr(stmt.Key, syntax.NameLoopVariable)
			scan.addExpr(stmt.Value, syntax.NameLoopVariable)
		}
	case *ast.ForStmt:
		if initStmt, ok := stmt.Init.(*ast.AssignStmt); ok {
			scan.define(initStmt, syntax.NameLoopVariable)
		}
	case *ast.TypeSwitchStmt:
		if assign, ok := stmt.Assign.(*ast.AssignStmt); ok {
			scan.define(assign, syntax.NameSwitchBinding)
		}
	case *ast.LabeledStmt:
		scan.add(stmt.Label, syntax.NameLabel)
	}
}

func (scan *collector) define(assign *ast.AssignStmt, kind syntax.NameKind) {
	if assign.Tok != token.DEFINE {
		return
	}
	for _, lhs := range assign.Lhs {
		scan.addExpr(lhs, kind)
	}
}

func (scan *collector) addExpr(expr ast.Expr, kind syntax.NameKind) {
	if ident, ok := expr.(*ast.Ident); ok {
		scan.add(ident, kind)
	}
}
