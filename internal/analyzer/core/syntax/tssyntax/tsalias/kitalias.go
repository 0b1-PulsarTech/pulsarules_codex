package tsalias

import (
	"path"
	"strings"
)

func (finder *Finder) kitAliases(root string) []Alias {
	const kitConfig = "svelte.config.js"
	body, ok := finder.read(path.Join(root, kitConfig))
	if !ok || len(body) > maxConfigBytes {
		return nil
	}
	text := string(body)
	for from := 0; ; {
		at := strings.Index(text[from:], "alias")
		if at < 0 {
			return nil
		}
		from += at + len("alias")
		if pairs, found := flatObject(text[from:]); found {
			return kitPairs(root, pairs)
		}
	}
}

type pair struct{ key, value string }

func flatObject(text string) ([]pair, bool) {
	scan := &literalScan{text: text}
	if !scan.expect(':') || !scan.expect('{') {
		return nil, false
	}
	var pairs []pair
	for {
		if scan.expect('}') {
			return pairs, true
		}
		key, ok := scan.literal()
		if !ok || !scan.expect(':') {
			return nil, false
		}
		value, ok := scan.literal()
		if !ok {
			return nil, false
		}
		pairs = append(pairs, pair{key, value})
		if !scan.expect(',') && !scan.peek('}') {
			return nil, false
		}
	}
}

func kitPairs(root string, pairs []pair) []Alias {
	var aliases []Alias
	for _, item := range pairs {
		target := path.Join(root, item.value)
		if target == ".." || strings.HasPrefix(target, "../") || item.key == "" {
			continue
		}
		if strings.HasSuffix(item.key, "*") {
			aliases = append(aliases, kitAlias(root, item.key, target))
			continue
		}
		aliases = append(aliases,
			kitAlias(root, item.key, target),
			kitAlias(root, item.key+"/*", target+"/*"),
		)
	}
	return aliases
}

func kitAlias(root, spec, target string) Alias {
	return Alias{Spec: spec, Target: target, Root: root, Origin: OriginKit}
}

type literalScan struct {
	text string
	pos  int
}

func (scan *literalScan) skipSpace() {
	for scan.pos < len(scan.text) && strings.IndexByte(" \t\r\n", scan.text[scan.pos]) >= 0 {
		scan.pos++
	}
}

func (scan *literalScan) peek(want byte) bool {
	scan.skipSpace()
	return scan.pos < len(scan.text) && scan.text[scan.pos] == want
}

func (scan *literalScan) expect(want byte) bool {
	if !scan.peek(want) {
		return false
	}
	scan.pos++
	return true
}

func (scan *literalScan) literal() (string, bool) {
	scan.skipSpace()
	if scan.pos >= len(scan.text) || strings.IndexByte("'\"", scan.text[scan.pos]) < 0 {
		return "", false
	}
	quote := scan.text[scan.pos]
	end := strings.IndexByte(scan.text[scan.pos+1:], quote)
	if end < 0 {
		return "", false
	}
	body := scan.text[scan.pos+1 : scan.pos+1+end]
	if strings.ContainsAny(body, "\\\n") {
		return "", false
	}
	scan.pos += end + 2
	return body, true
}
