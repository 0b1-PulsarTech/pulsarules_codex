package finding

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
	Symbol      string
}

type Reporter struct {
	analyzerID string
	severity   Severity
	category   Category
}

func NewReporter(analyzerID string, severity Severity, category Category) Reporter {
	return Reporter{analyzerID: analyzerID, severity: severity, category: category}
}

func (reporter Reporter) At(file string, line int, message, suggestion string) Finding {
	return Finding{
		AnalyzerID: reporter.analyzerID,
		Severity:   reporter.severity,
		Category:   reporter.category,
		File:       file,
		Line:       line,
		Message:    message,
		Suggestion: suggestion,
	}
}

func (reporter Reporter) New(message string) Finding {
	return Finding{
		AnalyzerID: reporter.analyzerID,
		Severity:   reporter.severity,
		Category:   reporter.category,
		Message:    message,
	}
}

func (reporter Reporter) NewWithSuggestion(message, suggestion string) Finding {
	finding := reporter.New(message)
	finding.Suggestion = suggestion
	return finding
}
