package finding

func SeverityByName() map[string]Severity {
	return map[string]Severity{
		"error":   SeverityError,
		"warning": SeverityWarning,
		"info":    SeverityInfo,
	}
}
