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

// UsageText returns the embedded usage banner, so a caller (e.g. a parity
// test asserting every dispatched command is documented) can inspect it
// without shelling out to Usage's stdout write.
func UsageText() string {
	return usageText
}
