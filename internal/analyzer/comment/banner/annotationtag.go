package banner

import "strings"

func annotationOnly(text string) bool {
	tagEnd := strings.IndexAny(text, " \t{")
	if tagEnd < 0 {
		return false
	}
	body := strings.TrimSpace(text[tagEnd:])
	tag := text[:tagEnd]
	if tag == "@import" {
		return isImportClause(body)
	}
	tail, ok := braceTail(body)
	if !ok {
		return false
	}
	switch tag {
	case "@type", "@satisfies":
		return tail == ""
	case "@typedef":
		return tail != "" && !strings.ContainsAny(tail, " \t")
	}
	return false
}

func braceTail(text string) (string, bool) {
	if !strings.HasPrefix(text, "{") {
		return "", false
	}
	depth := 0
	for idx := 0; idx < len(text); idx++ {
		switch text[idx] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return strings.TrimSpace(text[idx+1:]), true
			}
		}
	}
	return "", false
}

func isImportClause(clause string) bool {
	if strings.HasPrefix(clause, "{") {
		tail, _ := braceTail(clause)
		return isFromModule(strings.Fields(tail))
	}
	fields := strings.Fields(clause)
	split := len(fields) - 2
	if split < 1 || !isFromModule(fields[split:]) {
		return false
	}
	name := strings.TrimPrefix(strings.Join(fields[:split], " "), "* as ")
	return !strings.ContainsAny(name, " *")
}

func isFromModule(fields []string) bool {
	if len(fields) != 2 || fields[0] != "from" || len(fields[1]) < 2 {
		return false
	}
	module := fields[1]
	quote := module[0]
	return (quote == '\'' || quote == '"') && module[len(module)-1] == quote
}
