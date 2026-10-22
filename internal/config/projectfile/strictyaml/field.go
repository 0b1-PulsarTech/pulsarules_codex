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
	marked bool
	under  bool
}

func Root() *Field {
	// Interned: equal fields are one node, and a "." or "[" in a key never splits a segment.
	return &Field{}
}

func (at *Field) Key(name string) *Field {
	if name == "" {
		name = "<empty key>"
	}
	return at.child(segment{name: name})
}

func (at *Field) Item(idx int) *Field {
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

func (at *Field) mark() {
	at.marked = true
	for node := at; node.parent != nil; node = node.parent {
		if node.seg.indexed {
			node.parent.under = true
		}
	}
}

func (at *Field) covered(walked *int) bool {
	// Covered by an equal or enclosing problem, or when it holds an item with a problem.
	if at.marked || at.under {
		return true
	}
	for node := at.parent; node != nil; node = node.parent {
		*walked++
		if node.marked {
			return true
		}
	}
	return false
}

func ClipValue(value string) string {
	if len(value) <= maxClipBytes {
		return value
	}
	cut := maxClipBytes
	for cut > 0 && !utf8.RuneStart(value[cut]) {
		cut--
	}
	return value[:cut] + ellipsis
}
