package arch

import (
	"fmt"
	"go/parser"
	"go/token"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/gomod"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/source"
)

func discoverPackages(provider source.Provider, modulePath string) map[string][]string {
	pkgMap := make(map[string][]string)

	provider.Walk(func(file string, ext string) bool {
		if ext != ".go" || strings.HasSuffix(file, "_test.go") || inSkippedDir(file) {
			return true
		}
		src, ok := provider.Read(file)
		if !ok {
			return true
		}
		imports, err := parseImports(file, src)
		if err != nil {
			// One malformed file must not halt discovery of the rest.
			return true
		}

		dir := path.Dir(filepath.ToSlash(file))
		pkgImport := modulePath
		if dir != "." {
			pkgImport = modulePath + "/" + dir
		}
		known := pkgMap[pkgImport]
		for _, imp := range imports {
			if !slices.Contains(known, imp) {
				known = append(known, imp)
			}
		}
		pkgMap[pkgImport] = known
		return true
	})

	return pkgMap
}

func inSkippedDir(file string) bool {
	segments := strings.Split(filepath.ToSlash(file), "/")
	for _, segment := range segments[:len(segments)-1] {
		if gomod.IsHiddenName(segment) || segment == "vendor" || segment == "node_modules" {
			return true
		}
	}
	return false
}

func parseImports(file string, src []byte) ([]string, error) {
	parsed, err := parser.ParseFile(token.NewFileSet(), file, src, parser.ImportsOnly)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", file, err)
	}
	var imports []string
	for _, spec := range parsed.Imports {
		imports = append(imports, strings.Trim(spec.Path.Value, `"`))
	}
	return imports, nil
}

func localImports(modulePath string, imports []string) []string {
	prefix := modulePath + "/"
	var out []string
	for _, imp := range imports {
		if imp == modulePath || strings.HasPrefix(imp, prefix) {
			out = append(out, imp)
		}
	}
	return out
}
