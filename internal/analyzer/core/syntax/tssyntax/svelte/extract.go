package svelte

import "errors"

var (
	errUnterminated  = errors.New("unterminated script block")
	ErrDroppedScript = errors.New("a script block below the top level was not analysed")
	ErrUnclosed      = errors.New("an unclosed brace stopped the markup scan")
)

type Script struct {
	TypeScript bool
	Text       string
}

func Extract(text string) ([]Script, error) {
	scan := &markup{text: text, lower: asciiLower(text)}
	if err := scan.run(); err != nil {
		return nil, err
	}
	if scan.unclosed {
		return scan.scripts, ErrUnclosed
	}
	if scan.dropped {
		return scan.scripts, ErrDroppedScript
	}
	return scan.scripts, nil
}

func blank(text string, start, stop int) string {
	buf := make([]byte, len(text))
	for idx := range buf {
		buf[idx] = ' '
		if text[idx] == '\n' || (idx >= start && idx < stop) {
			buf[idx] = text[idx]
		}
	}
	return string(buf)
}

func asciiLower(text string) string {
	buf := []byte(text)
	for idx, char := range buf {
		if char >= 'A' && char <= 'Z' {
			buf[idx] = char + ('a' - 'A')
		}
	}
	return string(buf)
}
