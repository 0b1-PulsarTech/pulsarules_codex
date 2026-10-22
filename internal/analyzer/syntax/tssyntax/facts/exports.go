package facts

import (
	"github.com/Zzzen/typescript-go/use-at-your-own-risk/ast"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

func exportedLocals(statements []*ast.Node) map[string]bool {
	locals := map[string]bool{}
	for _, stmt := range statements {
		if stmt.Kind != ast.KindExportDeclaration {
			continue
		}
		clause := stmt.AsExportDeclaration()
		if clause.ModuleSpecifier != nil || clause.ExportClause == nil ||
			clause.ExportClause.Kind != ast.KindNamedExports {
			continue
		}
		for _, specifier := range clause.ExportClause.Elements() {
			if local := specifier.PropertyNameOrName(); local.Kind == ast.KindIdentifier {
				locals[local.Text()] = true
			}
		}
	}
	return locals
}

func (names *nameFacts) addExported(name *ast.Node, template syntax.Name) {
	before := len(names.out)
	names.addName(name, template)
	if len(names.out) > before {
		added := &names.out[before]
		added.Exported = names.exported(name, added.Name)
	}
}

func (names *nameFacts) exported(name *ast.Node, text string) bool {
	declaration := declarationOf(name)
	if declaration == nil || declaration.Parent == nil ||
		declaration.Parent.Kind != ast.KindSourceFile {
		return false
	}
	return ast.HasSyntacticModifier(declaration, ast.ModifierFlagsExport) || names.locals[text]
}

func declarationOf(name *ast.Node) *ast.Node {
	node := name.Parent
	for node != nil {
		switch node.Kind {
		case ast.KindBindingElement, ast.KindObjectBindingPattern, ast.KindArrayBindingPattern,
			ast.KindVariableDeclaration, ast.KindVariableDeclarationList:
			node = node.Parent
		default:
			return node
		}
	}
	return nil
}
