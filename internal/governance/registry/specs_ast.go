package registry

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/vacuousdoc"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/comment/testdoc"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/complexity"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/controlflow"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/declaration/shadowing"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/declaration/singleuse"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/signature/namedreturns"
)

func astSpecs() []analyzerSpec {
	return []analyzerSpec{
		staticSpec("control-flow", controlflow.NewAnalyzer),
		staticSpec("shadowing", shadowing.NewAnalyzer),
		staticSpec("complexity", complexity.NewAnalyzer),
		staticSpec("named-returns", namedreturns.NewAnalyzer),
		staticSpec("vacuous-doc", vacuousdoc.NewAnalyzer),
		staticSpec("test-comment", testdoc.NewAnalyzer),
		staticSpec("single-use-declaration", singleuse.NewAnalyzer),
	}
}
