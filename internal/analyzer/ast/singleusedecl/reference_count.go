package singleusedecl

import (
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

func checkCandidate(
	ctx *core.AnalysisContext,
	provider core.DeclarationProvider,
	decl core.Declaration,
	reporter core.Reporter,
) []core.Finding {
	owner, ok := provider.SoleUser(ctx, decl)
	if !ok {
		return nil
	}
	finding := reporter.At(
		decl.File,
		decl.Line,
		fmt.Sprintf("%s is referenced only inside %s", decl.Name, owner),
		fmt.Sprintf("move the declaration inside %s", owner),
	)
	finding.Symbol = decl.Name
	return []core.Finding{finding}
}
