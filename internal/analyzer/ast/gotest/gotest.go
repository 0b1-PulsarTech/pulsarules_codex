package gotest

import (
	"go/ast"
	"strings"
	"unicode"
)

const KindExample = "Example"

func testPrefixes() []string {
	return []string{"Test", "Benchmark", "Fuzz", KindExample}
}

func TestKind(file *ast.File, fn *ast.FuncDecl) (string, bool) {
	if fn.Recv != nil || fn.Name == nil {
		return "", false
	}
	alias := testingAlias(file)
	name := fn.Name.Name
	for _, prefix := range testPrefixes() {
		suffix, ok := strings.CutPrefix(name, prefix)
		if !ok {
			continue
		}
		if suffix != "" && unicode.IsLower(rune(suffix[0])) {
			continue
		}
		if matchesSignature(name, prefix, alias, fn.Type) {
			return prefix, true
		}
	}
	return "", false
}
