package goast

import "go/ast"

type Visitor struct {
	Block func(block *ast.BlockStmt, depth int)
	Else  func(block *ast.BlockStmt)
}

func WalkFile(file *ast.File, visitor Visitor) {
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			walkBlock(fn.Body, 0, visitor)
		}
	}
}

func walkBlock(block *ast.BlockStmt, depth int, visitor Visitor) {
	if block == nil {
		return
	}
	if visitor.Block != nil {
		visitor.Block(block, depth)
	}
	for _, stmt := range block.List {
		walkStmt(stmt, depth, visitor)
	}
}

func walkStmt(stmt ast.Stmt, depth int, visitor Visitor) {
	switch typed := stmt.(type) {
	case *ast.IfStmt:
		walkIf(typed, depth, visitor)
	case *ast.ForStmt:
		walkBlock(typed.Body, depth+1, visitor)
	case *ast.RangeStmt:
		walkBlock(typed.Body, depth+1, visitor)
	// A case clause is not a block statement, so case bodies are never walked.
	case *ast.SwitchStmt:
		walkBlock(typed.Body, depth+1, visitor)
	case *ast.TypeSwitchStmt:
		walkBlock(typed.Body, depth+1, visitor)
	case *ast.SelectStmt:
		walkBlock(typed.Body, depth+1, visitor)
	case *ast.BlockStmt:
		walkBlock(typed, depth+1, visitor)
	}
}

func walkIf(ifStmt *ast.IfStmt, depth int, visitor Visitor) {
	walkBlock(ifStmt.Body, depth+1, visitor)
	elseBlock, ok := ifStmt.Else.(*ast.BlockStmt)
	if !ok {
		if elseIf, isIf := ifStmt.Else.(*ast.IfStmt); isIf {
			walkIf(elseIf, depth, visitor)
		}
		return
	}
	if visitor.Else != nil {
		visitor.Else(elseBlock)
	}
	walkBlock(elseBlock, depth+1, visitor)
}
