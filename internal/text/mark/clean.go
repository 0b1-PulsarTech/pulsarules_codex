package mark

import "strings"

// Clean removes every ClassStrip marker and folds every ClassSpace one to a
// plain space, leaving ClassContextual and ClassTypographic untouched.
//
// why: guessing at emoji glue or at a dash inside a string literal would corrupt
// data the author meant to keep, after the write tool reported success.
func Clean(src string) (cleaned string, acted []Mark) {
	marks := Scan(src)
	if len(marks) == 0 {
		return src, nil
	}
	var out strings.Builder
	out.Grow(len(src))
	prev := 0
	for _, mark := range marks {
		if mark.Class != ClassStrip && mark.Class != ClassSpace {
			continue
		}
		out.WriteString(src[prev:mark.Offset])
		if mark.Class == ClassSpace {
			out.WriteByte(' ')
		}
		prev = mark.Offset + len(string(mark.Rune))
		acted = append(acted, mark)
	}
	if len(acted) == 0 {
		return src, nil
	}
	out.WriteString(src[prev:])
	return out.String(), acted
}
