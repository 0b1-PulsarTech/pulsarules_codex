package codex

import "strings"

type clashScan struct {
	name, table string
	inParent    bool
}

func findClash(text, name string) string {
	flatten := strings.NewReplacer(" ", "", "\t", "", `"`, "", "'", "")
	scan := clashScan{name: name, table: "mcp_servers." + name}
	for raw := range strings.SplitSeq(text, "\n") {
		if found := scan.line(flatten.Replace(strings.TrimSpace(raw))); found != "" {
			return found
		}
	}
	return ""
}

func (scan *clashScan) line(line string) string {
	switch {
	case line == "" || strings.HasPrefix(line, "#"):
		return ""
	case strings.HasPrefix(line, "["):
		return scan.header(line)
	case strings.HasPrefix(line, "mcp_servers="):
		// An inline table cannot be extended by a later [mcp_servers.x] header.
		return "an inline mcp_servers table, which cannot be extended"
	case hasKey(line, scan.table):
		return scan.table + " as a dotted key"
	case scan.inParent && hasKey(line, scan.name):
		return scan.name + " under [mcp_servers]"
	}
	return ""
}

func (scan *clashScan) header(line string) string {
	// A bare [mcp_servers] header only matters for the keys written under it.
	scan.inParent = line == "[mcp_servers]" || strings.HasPrefix(line, "[mcp_servers]#")
	if strings.HasPrefix(line, "["+scan.table+"]") || strings.HasPrefix(line, "["+scan.table+".") {
		return "[" + scan.table + "]"
	}
	return ""
}

func hasKey(line, key string) bool {
	return strings.HasPrefix(line, key+".") || strings.HasPrefix(line, key+"=")
}
