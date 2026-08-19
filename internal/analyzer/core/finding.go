package core

// Finding is a single issue reported by an analyzer. It carries enough context
// for the pipeline to inject the corresponding rule's distilled summary (via
// RuleSummary, filled in the StageRuleInjection stage).
type Finding struct {
	// AnalyzerID is the ID of the analyzer that produced this finding.
	AnalyzerID string
	// Severity controls whether the finding blocks the operation.
	Severity Severity
	// Category classifies the analysis technique.
	Category Category
	// File is the repo-relative path of the affected file, or empty for
	// findings not tied to a file (e.g. commit-message findings).
	File string
	// Line is the 1-based line number, or 0 if not applicable.
	Line int
	// Message is the human/agent-readable description of the issue.
	Message string
	// Suggestion is an optional fix suggestion, or empty if none.
	Suggestion string
	// RuleID is the knowledge-base rule ID that this finding corresponds to.
	// Used by StageRuleInjection to attach the rule's distilled summary.
	RuleID string
	// RuleSummary is the rule's distilled one-paragraph summary corresponding
	// to this finding, injected by the StageRuleInjection stage. Empty until
	// then.
	RuleSummary string
}

// Reporter stamps the AnalyzerID, Severity and Category that are constant for
// one analyzer onto every finding it emits, so a call site carries only what
// varies: where the problem is and what to say about it. It is a SHALLOW
// helper: it earns its place by weakening the connascence of a magic trio
// repeated at every call site, not by hiding any real complexity.
type Reporter struct {
	analyzerID string
	severity   Severity
	category   Category
}

func NewReporter(analyzerID string, severity Severity, category Category) Reporter {
	return Reporter{analyzerID: analyzerID, severity: severity, category: category}
}

// Resolved returns a copy of r with its severity overridden by ctx's
// ParamSeverity param for r's own analyzer id, when the run's config sets
// one; otherwise r keeps its own default. Most Reporter values are built
// once as a package-level var at init, frozen before any config exists;
// calling Resolved(ctx) at the top of Analyze is what lets a run's config
// reach a severity that would otherwise be a compile-time constant, without
// every analyzer spelling out the ctx.Params lookup itself.
func (r Reporter) Resolved(ctx *AnalysisContext) Reporter {
	r.severity = ctx.Params(r.analyzerID).Severity(r.severity)
	return r
}

// At builds a Finding tied to a specific file and line, such as one found
// while walking a source file's AST or text.
func (r Reporter) At(file string, line int, message, suggestion string) Finding {
	return Finding{
		AnalyzerID: r.analyzerID,
		Severity:   r.severity,
		Category:   r.category,
		File:       file,
		Line:       line,
		Message:    message,
		Suggestion: suggestion,
	}
}

// New builds a Finding with no file or line, for a check that is not tied to
// a location, such as a commit-message rule.
func (r Reporter) New(message string) Finding {
	return Finding{
		AnalyzerID: r.analyzerID,
		Severity:   r.severity,
		Category:   r.category,
		Message:    message,
	}
}

// NewWithSuggestion builds a Finding with no file or line but a fix
// suggestion attached.
func (r Reporter) NewWithSuggestion(message, suggestion string) Finding {
	f := r.New(message)
	f.Suggestion = suggestion
	return f
}
