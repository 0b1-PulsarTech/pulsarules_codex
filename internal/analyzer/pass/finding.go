package pass

import "github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"

func ResolveReporter(reporter finding.Reporter, ctx *Context) finding.Reporter {
	// Shared reporters are built once; severity is per-run policy, so each run resolves a copy.
	reporter.Severity = ctx.Params(reporter.AnalyzerID).Severity(reporter.Severity)
	return reporter
}
