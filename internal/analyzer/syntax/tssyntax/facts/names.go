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

func (facts *nameFacts) visit(node *ast.Node) {
	switch node.Kind {
	case ast.KindFunctionDeclaration:
		facts.addExported(node.Name(), syntax.Name{Kind: syntax.NameFunction})
	case ast.KindFunctionExpression:
		facts.addName(node.Name(), syntax.Name{Kind: syntax.NameFunction})
	case ast.KindMethodDeclaration, ast.KindMethodSignature, ast.KindGetAccessor,
		ast.KindSetAccessor:
		facts.addName(node.Name(), syntax.Name{Kind: syntax.NameMethod})
	case ast.KindClassDeclaration, ast.KindClassExpression, ast.KindInterfaceDeclaration,
		ast.KindTypeAliasDeclaration, ast.KindEnumDeclaration, ast.KindModuleDeclaration:
		facts.addExported(node.Name(), syntax.Name{Kind: syntax.NameType})
	case ast.KindEnumMember:
		facts.addName(node.Name(), syntax.Name{Kind: syntax.NameEnumValue})
	case ast.KindPropertyDeclaration, ast.KindPropertySignature:
		facts.addName(node.Name(), syntax.Name{Kind: syntax.NameField})
	case ast.KindTypeParameter:
		facts.addName(node.Name(), syntax.Name{Kind: syntax.NameTypeParam})
	case ast.KindParameter:
		facts.parameter(node)
	case ast.KindVariableDeclaration:
		facts.variable(node)
	default:
		facts.otherNames(node)
	}
}

func (facts *nameFacts) otherNames(node *ast.Node) {
	switch node.Kind {
	case ast.KindLabeledStatement:
		facts.addLabel(node.AsLabeledStatement().Label)
	case ast.KindImportClause, ast.KindNamespaceImport, ast.KindImportEqualsDeclaration:
		facts.addName(node.Name(), syntax.Name{Kind: syntax.NameImportAlias})
	case ast.KindImportSpecifier:
		if node.AsImportSpecifier().PropertyName != nil {
			facts.addName(node.Name(), syntax.Name{Kind: syntax.NameImportAlias})
		}
	}
}

func (facts *nameFacts) addName(name *ast.Node, template syntax.Name) {
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
	template.Line = facts.lineAt(name.Pos())
	facts.out = append(facts.out, template)
}

func (facts *nameFacts) addLabel(label *ast.Node) {
	facts.addName(label, syntax.Name{Kind: syntax.NameLabel})
	last := len(facts.out) - 1
	// A Svelte reactive statement is the label "$", fixed by the framework.
	if last >= 0 && facts.out[last].Name == "$" {
		facts.out[last].Exempt = true
	}
}

func (facts *nameFacts) parameter(node *ast.Node) {
	if !isThisParam(node) {
		facts.binding(node.Name(), syntax.Name{Kind: syntax.NameParameter})
	}
}

func (facts *nameFacts) variable(node *ast.Node) {
	facts.binding(node.Name(), variableKind(node))
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

func (facts *nameFacts) binding(name *ast.Node, template syntax.Name) {
	if name == nil {
		return
	}
	switch name.Kind {
	case ast.KindObjectBindingPattern, ast.KindArrayBindingPattern:
		for _, element := range name.AsBindingPattern().Elements.Nodes {
			if element.Kind == ast.KindBindingElement {
				facts.binding(element.Name(), template)
			}
		}
	default:
		facts.addExported(name, template)
	}
}
