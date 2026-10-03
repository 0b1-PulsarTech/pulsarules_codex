package movepurity

import (
	"strings"
)

func isProductionGo(file string) bool {
	return strings.HasSuffix(file, ".go") && !strings.HasSuffix(file, "_test.go")
}
