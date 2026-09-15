package testfunc

import (
	"go/ast"
	"strings"
	"unicode"
)

const KindExample = "Example"

func testPrefixes() []string {
	return []string{"Test", "Benchmark", "Fuzz", KindExample}
}

func Kind(fn *ast.FuncDecl) (string, bool) {
	if fn.Recv != nil || fn.Name == nil {
		return "", false
	}
	name := fn.Name.Name
	for _, prefix := range testPrefixes() {
		suffix, ok := strings.CutPrefix(name, prefix)
		if !ok {
			continue
		}
		if suffix != "" && unicode.IsLower(rune(suffix[0])) {
			continue
		}
		if matchesSignature(prefix, fn.Type) {
			return prefix, true
		}
	}
	return "", false
}
