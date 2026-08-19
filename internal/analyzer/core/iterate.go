package core

func EachChangedFile(
	ctx *AnalysisContext,
	eligible func(fc FileChange) bool,
	check func(fc FileChange, src []byte) []Finding,
) []Finding {
	if ctx.Sources == nil {
		return nil
	}
	var findings []Finding
	for _, fc := range ctx.ChangedFiles {
		if !eligible(fc) {
			continue
		}
		src, ok := ctx.Sources.Read(fc.Path)
		if !ok {
			continue
		}
		findings = append(findings, check(fc, src)...)
	}
	return findings
}
