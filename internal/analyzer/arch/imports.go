package arch

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// pkgImports holds the imports for one Go package, keyed by its import path.
type pkgImports struct {
	Dir     string // filesystem directory
	Imports []string
}

// discoverPackages walks the project directory, finds all .go files (excluding
// test files and vendor), parses their imports, and returns the package-level
// import map. The module path prefix is used to resolve local imports.
func discoverPackages(projectDir, modulePath string) map[string]*pkgImports {
	pkgMap := make(map[string]*pkgImports)

	_ = filepath.WalkDir(projectDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
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
			return nil
		}

		if _, exists := pkgMap[pkgImport]; !exists {
			pkgMap[pkgImport] = &pkgImports{
				Dir:     filepath.Dir(path),
				Imports: imports,
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

// localImports filters the import list to only those under the given module
// path prefix, and strips the prefix to produce package-relative paths.
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
