package arch

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type pkgImports struct {
	Dir     string
	Imports []string
}

func discoverPackages(projectDir, modulePath string) map[string]*pkgImports {
	pkgMap := make(map[string]*pkgImports)

	_ = filepath.WalkDir(projectDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil //nolint:nilerr // WalkDir callback: skip the bad entry, keep walking.
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") || d.Name() == "vendor" ||
				d.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		rel, _ := filepath.Rel(projectDir, path)
		pkgPath := filepath.Dir(rel)
		if pkgPath == "." {
			pkgPath = ""
		}
		pkgImport := modulePath
		if pkgPath != "" {
			pkgImport = modulePath + "/" + filepath.ToSlash(pkgPath)
		}

		imports, err := parseFileImports(path)
		if err != nil {
			return nil //nolint:nilerr // one malformed file must not halt discovery of the rest.
		}

		pkg, exists := pkgMap[pkgImport]
		if !exists {
			pkg = &pkgImports{Dir: filepath.Dir(path)}
			pkgMap[pkgImport] = pkg
		}
		for _, imp := range imports {
			if !slices.Contains(pkg.Imports, imp) {
				pkg.Imports = append(pkg.Imports, imp)
			}
		}
		return nil
	})

	return pkgMap
}

func parseFileImports(path string) ([]string, error) {
	//nolint:gosec // path from filepath.WalkDir, trusted
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	var imports []string
	for _, spec := range f.Imports {
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
