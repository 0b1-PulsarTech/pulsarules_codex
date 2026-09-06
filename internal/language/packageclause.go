package language

import "strings"

func isGoPackageDeclaration(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "package ")
}
