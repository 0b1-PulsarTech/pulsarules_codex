package strictyaml

import (
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

type segment struct {
	name    string
	index   int
	indexed bool
}

type Field struct {
	parent *Field
	seg    segment
	kids   map[segment]*Field
}

func newRoot() *Field {
	// Interned: equal fields are one node, and a "." or "[" in a key never splits a segment.
	return &Field{}
}

func (at *Field) key(name string) *Field {
	if name == "" {
		name = "<empty key>"
	}
	return at.child(segment{name: name})
}

func (at *Field) item(idx int) *Field {
	return at.child(segment{index: idx, indexed: true})
}

func (at *Field) child(seg segment) *Field {
	if next, found := at.kids[seg]; found {
		return next
	}
	if at.kids == nil {
		at.kids = map[segment]*Field{}
	}
	next := &Field{parent: at, seg: seg}
	at.kids[seg] = next
	return next
}

func (at *Field) String() string {
	var chain []segment
	for node := at; node.parent != nil; node = node.parent {
		chain = append(chain, node.seg)
	}
	var out strings.Builder
	for _, seg := range slices.Backward(chain) {
		switch {
		case seg.indexed:
			out.WriteString("[" + strconv.Itoa(seg.index) + "]")
		case out.Len() > 0:
			out.WriteString("." + seg.name)
		default:
			out.WriteString(seg.name)
		}
	}
	return out.String()
}

func clipValue(value string) string {
	const maxClipBytes = 80
	if len(value) <= maxClipBytes {
		return value
	}
	cut := maxClipBytes
	for cut > 0 && !utf8.RuneStart(value[cut]) {
		cut--
	}
	return value[:cut] + "..."
}
