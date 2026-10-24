package dartsrc

import (
	"path"
	"strings"
	"unicode"
)

type Roots map[string]string

func PackageName(pubspec string) string {
	for line := range strings.Lines(pubspec) {
		value, isName := strings.CutPrefix(line, "name:")
		if !isName {
			continue
		}
		value, _, _ = strings.Cut(value, " #")
		return strings.Trim(strings.TrimSpace(value), `"'`)
	}
	return ""
}

func BuildRoots(files []string, read func(file string) (string, error)) Roots {
	roots := Roots{}
	for _, file := range files {
		if path.Base(file) != "pubspec.yaml" {
			continue
		}
		content, err := read(file)
		if err != nil {
			continue
		}
		name := PackageName(content)
		if prior, isSeen := roots[name]; isSeen && prior != path.Dir(file) {
			roots[name] = "" // two pubspecs claim the name: no import through it can be followed
		} else if name != "" {
			roots[name] = path.Dir(file)
		}
	}
	return roots
}

func Resolve(importer, uri string, roots Roots) (target string, ok bool) {
	if rest, isPackage := strings.CutPrefix(uri, "package:"); isPackage {
		name, tail, _ := strings.Cut(rest, "/")
		dir := roots[name]
		if dir == "" || tail == "" {
			return "", false
		}
		return path.Join(dir, "lib", tail), true
	}
	scheme, _, _ := strings.Cut(uri, "/")
	if strings.Contains(scheme, ":") || strings.HasPrefix(uri, "/") {
		return "", false
	}
	target = path.Join(path.Dir(importer), uri)
	if target == ".." || strings.HasPrefix(target, "../") {
		return "", false
	}
	return target, true
}

func isConditionValue(before string) bool {
	return strings.HasSuffix(strings.TrimRight(before, " \t\r\n"), "==")
}

func literalValue(run string) (string, error) {
	body := strings.TrimPrefix(run, "r")
	if body == "" {
		return "", ErrMalformed
	}
	quote := body[:1]
	if quote != "'" && quote != `"` {
		return "", ErrMalformed
	}
	delim := quote
	if tripled := quote + quote + quote; strings.HasPrefix(body, tripled) {
		delim = tripled
	}
	if len(body) < 2*len(delim) || !strings.HasSuffix(body, delim) {
		return "", ErrMalformed
	}
	content := body[len(delim) : len(body)-len(delim)]
	if strings.ContainsAny(content, quote+`\$%?#`+"\n") {
		return "", ErrMalformed
	}
	return content, checkPackagePath(content)
}

func checkPackagePath(uri string) error {
	rest, isPackage := strings.CutPrefix(uri, "package:")
	if !isPackage {
		return nil
	}
	_, tail, _ := strings.Cut(rest, "/")
	for segment := range strings.SplitSeq(tail, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return ErrMalformed
		}
	}
	return nil
}

func (scan *scanner) isPlaced(result Directive, from, at int) bool {
	var code strings.Builder
	for offset := from; offset < at; offset++ {
		isTrivia := scan.lits.Contains(offset) || scan.lits.InComment(offset)
		if !isTrivia && !unicode.IsSpace(rune(scan.src[offset])) {
			code.WriteByte(scan.src[offset])
		}
	}
	if len(result.URIs) == 0 {
		return code.String() == strings.ReplaceAll(result.Kind, " ", "")
	}
	return strings.HasPrefix(code.String(), "if(") && strings.HasSuffix(code.String(), ")")
}
