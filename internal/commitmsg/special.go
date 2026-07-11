package commitmsg

import "strings"

func (p *parser) isSpecialInitial(msg *Message) bool {
	if len(msg.Emojis) == 1 && msg.Emojis[0] == "ghost" {
		rest := strings.TrimSpace(string(p.runes[p.pos:]))
		return strings.HasPrefix(rest, "Initial Commit") ||
			strings.HasPrefix(rest, "Initial commit")
	}
	return false
}

func (p *parser) isSpecialMerge(msg *Message) bool {
	if len(msg.Emojis) >= 1 && msg.Emojis[0] == "volcano" {
		rest := strings.TrimSpace(string(p.runes[p.pos:]))
		return strings.HasPrefix(rest, "Merge ")
	}
	return false
}

func isFooterLine(line string) bool {
	if strings.HasPrefix(line, "Co-Authored-By:") || strings.HasPrefix(line, "Claude-Session:") {
		return true
	}
	if strings.HasPrefix(line, "BREAKING CHANGE:") || strings.HasPrefix(line, "BREAKING-CHANGE:") {
		return true
	}
	if strings.HasPrefix(line, "Closes:") || strings.HasPrefix(line, "Refs:") {
		return true
	}
	return false
}

func parseFooter(line string) Footer {
	for _, sep := range []string{": ", " #"} {
		if idx := strings.Index(line, sep); idx > 0 {
			return Footer{
				Key:   strings.TrimSpace(line[:idx]),
				Value: strings.TrimSpace(line[idx+len(sep):]),
			}
		}
	}
	return Footer{}
}
