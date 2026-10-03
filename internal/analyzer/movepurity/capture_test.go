package movepurity

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

const (
	routingMain = "internal/routing/r.go"
	useFoo      = "func F() int { return Foo() }\n"
)

func pkgFile(pkg, body string) string { return "package " + pkg + "\n\n" + body }

func moveF(extra ...core.Rename) []core.Rename {
	return append([]core.Rename{pair(utilA, routingA)}, extra...)
}
