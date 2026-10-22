package facts

import (
	"github.com/Zzzen/typescript-go/use-at-your-own-risk/ast"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

type nameFacts struct {
	source
	locals map[string]bool
	out    []syntax.Name
}

func (names *nameFacts) visit(node *ast.Node) {
	switch node.Kind {
	case ast.KindFunctionDeclaration:
		names.addExported(node.Name(), syntax.Name{Kind: syntax.NameFunction})
	case ast.KindFunctionExpression:
		names.addName(node.Name(), syntax.Name{Kind: syntax.NameFunction})
	case ast.KindMethodDeclaration, ast.KindMethodSignature, ast.KindGetAccessor,
		ast.KindSetAccessor:
		names.addName(node.Name(), syntax.Name{Kind: syntax.NameMethod})
	case ast.KindClassDeclaration, ast.KindClassExpression, ast.KindInterfaceDeclaration,
		ast.KindTypeAliasDeclaration, ast.KindEnumDeclaration, ast.KindModuleDeclaration:
		names.addExported(node.Name(), syntax.Name{Kind: syntax.NameType})
	case ast.KindEnumMember:
		names.addName(node.Name(), syntax.Name{Kind: syntax.NameEnumValue})
	case ast.KindPropertyDeclaration, ast.KindPropertySignature:
		names.addName(node.Name(), syntax.Name{Kind: syntax.NameField})
	case ast.KindTypeParameter:
		names.addName(node.Name(), syntax.Name{Kind: syntax.NameTypeParam})
	case ast.KindParameter:
		names.parameter(node)
	case ast.KindVariableDeclaration:
		names.variable(node)
	default:
		names.otherNames(node)
	}
}

func (names *nameFacts) otherNames(node *ast.Node) {
	switch node.Kind {
	case ast.KindLabeledStatement:
		names.addLabel(node.AsLabeledStatement().Label)
	case ast.KindImportClause, ast.KindNamespaceImport, ast.KindImportEqualsDeclaration:
		names.addName(node.Name(), syntax.Name{Kind: syntax.NameImportAlias})
	case ast.KindImportSpecifier:
		if node.AsImportSpecifier().PropertyName != nil {
			names.addName(node.Name(), syntax.Name{Kind: syntax.NameImportAlias})
		}
	}
}

func (names *nameFacts) addName(name *ast.Node, template syntax.Name) {
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
	template.Line = names.lineAt(name.Pos())
	names.out = append(names.out, template)
}

func (names *nameFacts) addLabel(label *ast.Node) {
	names.addName(label, syntax.Name{Kind: syntax.NameLabel})
	last := len(names.out) - 1
	// A Svelte reactive statement is the label "$", fixed by the framework.
	if last >= 0 && names.out[last].Name == "$" {
		names.out[last].Exempt = true
	}
}

func (names *nameFacts) parameter(node *ast.Node) {
	if !isThisParam(node) {
		names.binding(node.Name(), syntax.Name{Kind: syntax.NameParameter})
	}
}

func (names *nameFacts) variable(node *ast.Node) {
	names.binding(node.Name(), variableKind(node))
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

func (names *nameFacts) binding(name *ast.Node, template syntax.Name) {
	if name == nil {
		return
	}
	switch name.Kind {
	case ast.KindObjectBindingPattern, ast.KindArrayBindingPattern:
		for _, element := range name.AsBindingPattern().Elements.Nodes {
			if element.Kind == ast.KindBindingElement {
				names.binding(element.Name(), template)
			}
		}
	default:
		names.addExported(name, template)
	}
}
