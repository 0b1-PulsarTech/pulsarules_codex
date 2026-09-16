package singleuse

import (
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

func checkCandidate(
	ctx *pass.Context,
	provider DeclarationProvider,
	decl Declaration,
	reporter pass.Reporter,
) []pass.Finding {
	owner, ok := provider.SoleUser(ctx, decl)
	if !ok {
		return nil
	}
	return []pass.Finding{reporter.At(
		decl.File,
		decl.Line,
		fmt.Sprintf("%s is referenced only inside %s", decl.Name, owner),
		fmt.Sprintf("move the declaration inside %s", owner),
	)}
}
