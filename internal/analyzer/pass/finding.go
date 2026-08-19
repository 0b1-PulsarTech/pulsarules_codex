package pass

type Finding struct {
	AnalyzerID  string
	Severity    Severity
	Category    Category
	File        string
	Line        int
	Message     string
	Suggestion  string
	RuleID      string
	RuleSummary string
}

type Reporter struct {
	analyzerID string
	severity   Severity
	category   Category
}

func NewReporter(analyzerID string, severity Severity, category Category) Reporter {
	return Reporter{analyzerID: analyzerID, severity: severity, category: category}
}

func ResolveReporter(r Reporter, ctx *Context) Reporter {
	// Shared reporters are built once; severity is per-run policy, so each run resolves a copy.
	r.severity = ctx.Params(r.analyzerID).Severity(r.severity)
	return r
}

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

func (r Reporter) New(message string) Finding {
	return Finding{
		AnalyzerID: r.analyzerID,
		Severity:   r.severity,
		Category:   r.category,
		Message:    message,
	}
}

func (r Reporter) NewWithSuggestion(message, suggestion string) Finding {
	f := r.New(message)
	f.Suggestion = suggestion
	return f
}
