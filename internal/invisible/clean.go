package invisible

import "strings"

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
