package loader

import "strings"

const adviceMark = "safe.directory "

func withoutRoot(err error, root string) string {
	// Git echoes the absolute root, which could fill the line before the reason.
	text := err.Error()
	if len(root) <= 1 {
		return text
	}
	lines := strings.Split(text, "\n")
	for idx, line := range lines {
		// The text after the safe.directory mark is a command to run, so its path stays real.
		head, tail := line, ""
		if cut := adviceStart(line); cut >= 0 {
			head, tail = line[:cut+len(adviceMark)], line[cut+len(adviceMark):]
		}
		lines[idx] = replaceRoot(head, root) + tail
	}
	return strings.Join(lines, "\n")
}

func adviceStart(line string) int {
	for from := 0; from < len(line); {
		found := strings.Index(line[from:], adviceMark)
		if found < 0 {
			break
		}
		cut := from + found
		// A mark right after a slash is the last directory name of the root.
		if cut == 0 || line[cut-1] != '/' {
			return cut
		}
		from = cut + len(adviceMark)
	}
	return -1
}

func replaceRoot(text, root string) string {
	if root == "" {
		return text
	}
	var out strings.Builder
	last := 0
	for from := 0; from < len(text); {
		found := strings.Index(text[from:], root)
		if found < 0 {
			break
		}
		start, end := from+found, from+found+len(root)
		from = max(end, start+1)
		if !startsPath(text[:start]) || !endsPath(text[end:]) {
			continue
		}
		out.WriteString(text[last:start] + ".")
		last = end
	}
	return out.String() + text[last:]
}

func startsPath(before string) bool {
	return before == "" || strings.IndexByte(" \t\"'`=:(", before[len(before)-1]) >= 0
}

func endsPath(rest string) bool {
	return rest == "" || strings.IndexByte("/\"'`:\t\r ", rest[0]) >= 0
}
