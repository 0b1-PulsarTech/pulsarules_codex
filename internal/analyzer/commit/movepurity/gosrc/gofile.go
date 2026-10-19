package gosrc

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"strconv"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/gomod"
)

type (
	goToken struct {
		kind token.Token
		lit  string
		off  int
	}
	ImportSpec struct {
		Name string
		Path string
	}
	File struct {
		Pkg        string
		preamble   []goToken
		docStart   int
		docEnd     int
		Header     []string
		body       []goToken
		Imports    []ImportSpec
		unresolved map[int]bool
		keys       map[int]bool
		examples   [][2]int
		TopLevel   map[string]bool
		Loads      bool
		Embeds     []string
		IsIgnored  bool
	}
)

func Parse(src string) (File, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		return File{}, fmt.Errorf("parse go source: %w", err)
	}
	tokFile := fset.File(file.Package)
	out := File{
		Pkg:        file.Name.Name,
		unresolved: map[int]bool{},
		keys:       map[int]bool{},
		docStart:   -1,
		docEnd:     -1,
		IsIgnored:  gomod.IsBuildIgnored(file),
		TopLevel:   map[string]bool{},
		Embeds:     embedPatterns(file),
		Imports:    importSpecs(file),
	}
	topLevelNames(file, out.TopLevel)
	out.examples = exampleBodies(file, tokFile)
	out.Loads = loadsAtInit(file, out.TopLevel)
	out.indexNames(file, tokFile)
	if file.Doc != nil {
		out.docStart, out.docEnd = tokFile.Offset(file.Doc.Pos()), tokFile.Offset(file.Doc.End())
	}
	out.split(
		src,
		tokFile.Offset(file.Package),
		tokFile.Offset(file.Name.End()),
		headerEnd(file, tokFile),
	)
	return out, nil
}

func (goFile *File) indexNames(file *ast.File, tokFile *token.File) {
	//nolint:staticcheck // Only the parser's name resolution separates local from package-level.
	for _, ident := range file.Unresolved {
		goFile.unresolved[tokFile.Offset(ident.Pos())] = true
	}
	// Per go.dev/issue/45160, the parser never lists a composite-literal key ident as unresolved.
	ast.Inspect(file, func(node ast.Node) bool {
		if pair, ok := node.(*ast.KeyValueExpr); ok {
			if key, isIdent := pair.Key.(*ast.Ident); isIdent &&
				key.Obj == nil { //nolint:staticcheck // see above
				goFile.keys[tokFile.Offset(key.Pos())] = true
			}
		}
		return true
	})
}

func importSpecs(file *ast.File) []ImportSpec {
	specs := make([]ImportSpec, 0, len(file.Imports))
	for _, spec := range file.Imports {
		unquoted, _ := strconv.Unquote(spec.Path.Value)
		name := ""
		if spec.Name != nil {
			name = spec.Name.Name
		}
		specs = append(specs, ImportSpec{Name: name, Path: unquoted})
	}
	return specs
}

func headerEnd(file *ast.File, tokFile *token.File) int {
	end := tokFile.Offset(file.Name.End())
	for _, decl := range file.Decls {
		if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.IMPORT {
			end = max(end, tokFile.Offset(gen.End()))
		}
	}
	return end
}

func (goFile *File) split(src string, pkgOff, nameEnd, bodyOff int) {
	var sc scanner.Scanner
	scanFile := token.NewFileSet().AddFile("", -1, len(src))
	sc.Init(scanFile, []byte(src), nil, scanner.ScanComments)
	for {
		pos, kind, lit := sc.Scan()
		if kind == token.EOF {
			goFile.body = dropOptionalSemicolons(goFile.body)
			return
		}
		off := scanFile.Offset(pos)
		if kind == token.COMMENT && off >= nameEnd && isImportComment(src, nameEnd, off, lit) {
			continue
		}
		goFile.place(goToken{kind: kind, lit: lit, off: off}, pkgOff, nameEnd, bodyOff)
	}
}

func (goFile *File) place(tok goToken, pkgOff, nameEnd, bodyOff int) {
	switch {
	case tok.off < pkgOff:
		goFile.preamble = append(goFile.preamble, tok)
	case tok.off < nameEnd:
	case tok.off < bodyOff:
		if tok.kind == token.COMMENT {
			goFile.Header = append(goFile.Header, tok.lit)
		}
	default:
		if tok.kind == token.SEMICOLON {
			tok.lit = ""
		}
		goFile.body = append(goFile.body, tok)
	}
}

func isImportComment(src string, nameEnd, off int, lit string) bool {
	if strings.Contains(src[nameEnd:off], "\n") {
		return false
	}
	return strings.HasPrefix(lit, "// import ") || strings.HasPrefix(lit, "/* import ")
}

func dropOptionalSemicolons(body []goToken) []goToken {
	kept := body[:0:0]
	for idx, tok := range body {
		// A semicolon before a closer is optional; a reflow moving the closer is no edit.
		if tok.kind == token.SEMICOLON && idx+1 < len(body) &&
			(body[idx+1].kind == token.RBRACE || body[idx+1].kind == token.RPAREN) {
			continue
		}
		kept = append(kept, tok)
	}
	return kept
}
