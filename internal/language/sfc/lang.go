package sfc

import "strings"

type attribute struct{ name, value string }

func parseAttrs(attrs string) []attribute {
	var found []attribute
	rest := strings.TrimLeft(attrs, blanks)
	for rest != "" {
		end := strings.IndexAny(rest, "= \t\r\n/")
		if end < 0 {
			return append(found, attribute{name: rest})
		}
		if end == 0 {
			rest = strings.TrimLeft(rest[1:], blanks)
			continue
		}
		attr := attribute{name: rest[:end]}
		rest = strings.TrimLeft(rest[end:], blanks)
		if strings.HasPrefix(rest, "=") {
			raw := strings.TrimLeft(rest[1:], blanks)
			attr.value = unquote(raw)
			rest = strings.TrimLeft(raw[valueWidth(raw):], blanks)
		}
		found = append(found, attr)
	}
	return found
}

const blanks = " \t\r\n"

func valueWidth(raw string) int {
	if raw == "" {
		return 0
	}
	if quote := raw[0]; quote == '"' || quote == '\'' {
		end := strings.IndexByte(raw[1:], quote)
		if end < 0 {
			return len(raw)
		}
		return end + 2
	}
	if raw[0] == '{' {
		if braces := skipBraces(raw, 0); braces.Closed {
			return braces.End
		}
		return 1
	}
	end := strings.IndexAny(raw, blanks+"/>")
	if end < 0 {
		return len(raw)
	}
	return end
}

func unquote(raw string) string {
	if raw == "" {
		return ""
	}
	if quote := raw[0]; quote == '"' || quote == '\'' {
		end := strings.IndexByte(raw[1:], quote)
		if end < 0 {
			return raw[1:]
		}
		return raw[1 : 1+end]
	}
	return raw[:valueWidth(raw)]
}

func attrValue(attrs, name string) (string, bool) {
	for _, attr := range parseAttrs(attrs) {
		if attr.name == name {
			return attr.value, true
		}
	}
	return "", false
}

func isTypeScript(attrs string) bool {
	for _, name := range []string{"lang", "type"} {
		value, found := attrValue(attrs, name)
		if found && isTypeScriptName(value) {
			return true
		}
	}
	return false
}

func isTypeScriptName(value string) bool {
	return value == "ts" || value == "typescript" || strings.HasSuffix(value, "/typescript")
}

func isJavaScript(attrs string) bool {
	value, found := attrValue(attrs, "type")
	if !found || value == "" || value == "module" || value == "ts" || value == "js" {
		return true
	}
	return strings.Contains(value, "javascript") || strings.Contains(value, "ecmascript") ||
		strings.Contains(value, "typescript") || value == "text/babel"
}

func isModule(attrs string) bool {
	if value, found := attrValue(attrs, "context"); found && value == "module" {
		return true
	}
	_, found := attrValue(attrs, "module")
	return found
}
