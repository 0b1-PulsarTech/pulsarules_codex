package tsfacts

import (
	"github.com/Zzzen/typescript-go/use-at-your-own-risk/ast"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax"
)

func (coll *collector) names(node *ast.Node) {
	switch node.Kind {
	case ast.KindFunctionDeclaration:
		coll.addExported(node.Name(), syntax.Name{Kind: syntax.NameFunction})
	case ast.KindFunctionExpression:
		coll.addName(node.Name(), syntax.Name{Kind: syntax.NameFunction})
	case ast.KindMethodDeclaration, ast.KindMethodSignature, ast.KindGetAccessor,
		ast.KindSetAccessor:
		coll.addName(node.Name(), syntax.Name{Kind: syntax.NameMethod})
	case ast.KindClassDeclaration, ast.KindClassExpression, ast.KindInterfaceDeclaration,
		ast.KindTypeAliasDeclaration, ast.KindEnumDeclaration, ast.KindModuleDeclaration:
		coll.addExported(node.Name(), syntax.Name{Kind: syntax.NameType})
	case ast.KindEnumMember:
		coll.addName(node.Name(), syntax.Name{Kind: syntax.NameEnumValue})
	case ast.KindPropertyDeclaration, ast.KindPropertySignature:
		coll.addName(node.Name(), syntax.Name{Kind: syntax.NameField})
	case ast.KindTypeParameter:
		coll.addName(node.Name(), syntax.Name{Kind: syntax.NameTypeParam})
	case ast.KindParameter:
		coll.parameter(node)
	case ast.KindVariableDeclaration:
		coll.variable(node)
	default:
		coll.otherNames(node)
	}
}

func (coll *collector) otherNames(node *ast.Node) {
	switch node.Kind {
	case ast.KindLabeledStatement:
		coll.addLabel(node.AsLabeledStatement().Label)
	case ast.KindImportClause, ast.KindNamespaceImport, ast.KindImportEqualsDeclaration:
		coll.addName(node.Name(), syntax.Name{Kind: syntax.NameImportAlias})
	case ast.KindImportSpecifier:
		if node.AsImportSpecifier().PropertyName != nil {
			coll.addName(node.Name(), syntax.Name{Kind: syntax.NameImportAlias})
		}
	}
}

func (coll *collector) addName(name *ast.Node, template syntax.Name) {
	if name == nil {
		return
	}
	if name.Kind != ast.KindIdentifier && name.Kind != ast.KindPrivateIdentifier {
		return
	}
	text := name.Text()
	if text == "" {
		return
	}
	template.Name = text
	template.Line = coll.lineAt(name.Pos())
	coll.out.Names = append(coll.out.Names, template)
}

func (coll *collector) addLabel(label *ast.Node) {
	coll.addName(label, syntax.Name{Kind: syntax.NameLabel})
	last := len(coll.out.Names) - 1
	// A Svelte reactive statement is the label "$", fixed by the framework.
	if last >= 0 && coll.out.Names[last].Name == "$" {
		coll.out.Names[last].Exempt = true
	}
}

func (coll *collector) parameter(node *ast.Node) {
	if !isThisParam(node) {
		coll.binding(node.Name(), syntax.Name{Kind: syntax.NameParameter})
	}
}

func (coll *collector) variable(node *ast.Node) {
	coll.binding(node.Name(), variableKind(node))
}

func variableKind(node *ast.Node) syntax.Name {
	list := node.Parent
	if list == nil || list.Kind == ast.KindCatchClause {
		return syntax.Name{Kind: syntax.NameCatchParameter}
	}
	owner := list.Parent
	if owner == nil {
		return syntax.Name{Kind: syntax.NameVariable}
	}
	switch owner.Kind {
	case ast.KindForStatement, ast.KindForInStatement, ast.KindForOfStatement:
		return syntax.Name{Kind: syntax.NameLoopVariable}
	}
	if list.Flags&ast.NodeFlagsConst != 0 {
		return syntax.Name{Kind: syntax.NameConstant}
	}
	return syntax.Name{Kind: syntax.NameVariable}
}

func (coll *collector) binding(name *ast.Node, template syntax.Name) {
	if name == nil {
		return
	}
	switch name.Kind {
	case ast.KindObjectBindingPattern, ast.KindArrayBindingPattern:
		for _, element := range name.AsBindingPattern().Elements.Nodes {
			if element.Kind == ast.KindBindingElement {
				coll.binding(element.Name(), template)
			}
		}
	default:
		coll.addExported(name, template)
	}
}
