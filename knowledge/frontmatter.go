package knowledge

import (
	"errors"
	"strings"
)

func splitFrontmatter(raw []byte) (meta []byte, body string, err error) {
	const fence = "---"

	content := string(raw)
	if !strings.HasPrefix(content, fence+"\n") {
		return nil, "", errors.New("missing opening frontmatter fence")
	}
	rest := content[len(fence)+1:]

	// The common case: a closing fence on its own line, with a body after it.
	closeIdx := strings.Index(rest, "\n"+fence+"\n")
	if closeIdx >= 0 {
		return []byte(rest[:closeIdx]), rest[closeIdx+len("\n"+fence)+1:], nil
	}
	// No body: the rest ends with a closing fence.
	header, ok := strings.CutSuffix(rest, "\n"+fence)
	if !ok {
		return nil, "", errors.New("missing closing frontmatter fence")
	}
	return []byte(header), "", nil
}
