package finding

type Finding struct {
	AnalyzerID string
	Severity   Severity
	Category   Category
	File       string
	Line       int
	Message    string
	Suggestion string
	RuleID     string
	RuleBody   string
}

type Reporter struct {
	analyzerID string
	severity   Severity
	category   Category
}

func NewReporter(analyzerID string, severity Severity, category Category) Reporter {
	return Reporter{analyzerID: analyzerID, severity: severity, category: category}
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
