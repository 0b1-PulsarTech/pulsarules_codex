package analysis

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/complexity"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/controlflow"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/namedreturns"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/shadowing"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/singleusedecl"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/testcomment"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/vacuousdoc"
)

func astSpecs() []analyzerSpec {
	return []analyzerSpec{
		staticSpec("control-flow", controlflow.NewAnalyzer),
		staticSpec("shadowing", shadowing.NewAnalyzer),
		staticSpec("complexity", complexity.NewAnalyzer),
		staticSpec("named-returns", namedreturns.NewAnalyzer),
		staticSpec("vacuous-doc", vacuousdoc.NewAnalyzer),
		staticSpec("test-comment", testcomment.NewAnalyzer),
		staticSpec("single-use-declaration", singleusedecl.NewAnalyzer),
	}
}
