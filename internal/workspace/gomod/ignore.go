package gomod

import (
	"go/ast"
	"go/build/constraint"
	"strings"
)

func IsBuildIgnored(file *ast.File) bool {
	hasGoBuild, plusIgnored, plusOpen := false, false, true
	for _, group := range file.Comments {
		if group.Pos() >= file.Package {
			break
		}
		// go/build reads "// +build" lines only above the first block comment.
		if hasBlockComment(group) {
			plusOpen = false
		}
		for _, comment := range group.List {
			switch {
			case constraint.IsGoBuild(comment.Text):
				hasGoBuild = true
				if isIgnoreLine(comment.Text) {
					return true
				}
			// A "// +build" line counts only outside the package doc.
			case plusOpen && group != file.Doc && constraint.IsPlusBuild(comment.Text):
				plusIgnored = plusIgnored || isIgnoreLine(comment.Text)
			}
		}
	}
	// The +build rule applies only without a //go:build line.
	return plusIgnored && !hasGoBuild
}

func hasBlockComment(group *ast.CommentGroup) bool {
	for _, comment := range group.List {
		if strings.HasPrefix(comment.Text, "/*") {
			return true
		}
	}
	return false
}

func isIgnoreLine(text string) bool {
	expr, err := constraint.Parse(text)
	return err == nil && forcesIgnore(expr)
}

func forcesIgnore(expr constraint.Expr) bool {
	// True when the expression is false whenever "ignore" is unset.
	switch typed := expr.(type) {
	case *constraint.TagExpr:
		return typed.Tag == "ignore"
	case *constraint.NotExpr:
		return negationForcesIgnore(typed.X)
	case *constraint.AndExpr:
		return forcesIgnore(typed.X) || forcesIgnore(typed.Y)
	case *constraint.OrExpr:
		return forcesIgnore(typed.X) && forcesIgnore(typed.Y)
	}
	return false
}

func negationForcesIgnore(expr constraint.Expr) bool {
	switch typed := expr.(type) {
	case *constraint.NotExpr:
		return forcesIgnore(typed.X)
	case *constraint.AndExpr:
		return negationForcesIgnore(typed.X) && negationForcesIgnore(typed.Y)
	case *constraint.OrExpr:
		return negationForcesIgnore(typed.X) || negationForcesIgnore(typed.Y)
	}
	return false
}
