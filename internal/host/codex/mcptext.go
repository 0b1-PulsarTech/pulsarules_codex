package codex

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/mcpserver"
)

const (
	blockBegin = "# >>> pulsarules_cli managed block: removed by uninstall"
	blockEnd   = "# <<< pulsarules_cli managed block"
)

var ErrUnterminatedBlock = errors.New("managed block has no end marker")

func locate(lines []string) (start, end int, err error) {
	start = -1
	for idx, line := range lines {
		switch {
		case start < 0 && strings.TrimSpace(line) == blockBegin:
			start = idx
		case start >= 0 && strings.TrimSpace(line) == blockEnd:
			return start, idx + 1, nil
		}
	}
	if start >= 0 {
		return start, 0, ErrUnterminatedBlock
	}
	return -1, 0, nil
}

func blockBounds(lines []string) (start, end int, err error) {
	start, end, err = locate(lines)
	if err != nil || start < 0 {
		return start, end, err
	}
	// The blank line the writer put before the block goes with it.
	if start > 0 && strings.TrimSpace(lines[start-1]) == "" {
		start--
	}
	return start, end, nil
}

func renderBlock(servers []mcpserver.Server) string {
	if len(servers) == 0 {
		return ""
	}
	var out strings.Builder
	out.WriteString(blockBegin + "\n")
	for _, srv := range servers {
		quoted := make([]string, len(srv.Args))
		for idx, arg := range srv.Args {
			quoted[idx] = tomlString(arg)
		}
		fmt.Fprintf(&out, "[mcp_servers.%s]\ncommand = %s\nargs = [%s]\n",
			tomlKey(srv.Name), tomlString(srv.Command), strings.Join(quoted, ", "))
	}
	out.WriteString(blockEnd + "\n")
	return out.String()
}

func tomlKey(name string) string {
	for _, char := range name {
		isBare := char == '_' || char == '-' ||
			char >= '0' && char <= '9' || char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z'
		if !isBare {
			return tomlString(name)
		}
	}
	return name
}

func tomlString(value string) string {
	var out strings.Builder
	out.WriteByte('"')
	for _, char := range value {
		switch {
		case char == '"' || char == '\\':
			out.WriteByte('\\')
			out.WriteRune(char)
		case char == '\n':
			out.WriteString(`\n`)
		case char == '\t':
			out.WriteString(`\t`)
		case char == '\r':
			out.WriteString(`\r`)
		case unicode.IsControl(char):
			fmt.Fprintf(&out, `\u%04X`, char)
		default:
			out.WriteRune(char)
		}
	}
	out.WriteByte('"')
	return out.String()
}

type blockResult struct {
	text    string
	wired   []string
	clashed []string
}

func applyBlock(text string, servers []mcpserver.Server) (blockResult, error) {
	lines := strings.SplitAfter(text, "\n")
	start, end, err := blockBounds(lines)
	if err != nil {
		return blockResult{text: text}, err
	}
	head, tail := text, ""
	if start >= 0 {
		head, tail = strings.Join(lines[:start], ""), strings.Join(lines[end:], "")
	}
	var result blockResult
	var keep []mcpserver.Server
	for _, srv := range servers {
		if findClash(head+tail, srv.Name) != "" {
			result.clashed = append(result.clashed, srv.Name)
			continue
		}
		keep = append(keep, srv)
		result.wired = append(result.wired, srv.Name)
	}
	result.text = head + spaced(head, renderBlock(keep)) + tail
	return result, nil
}

func spaced(before, block string) string {
	if block == "" || before == "" {
		return block
	}
	if strings.HasSuffix(before, "\n") {
		return "\n" + block
	}
	return "\n\n" + block
}
