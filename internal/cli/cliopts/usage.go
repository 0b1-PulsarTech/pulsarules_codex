package cliopts

import (
	_ "embed"
	"fmt"
	"os"
)

//go:embed usage.txt
var usageText string

func Usage() error {
	_, _ = fmt.Fprint(os.Stdout, usageText)
	return nil
}

// UsageText returns the embedded usage banner.
// why: the dispatch-parity test reads the banner directly rather than
// capturing Usage's write to os.Stdout.
func UsageText() string {
	return usageText
}
