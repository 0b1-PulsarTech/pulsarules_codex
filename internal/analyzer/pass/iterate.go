package pass

func EachChangedFile(
	ctx *Context,
	eligible func(fc FileChange) bool,
	check func(fc FileChange, src []byte) []Finding,
) []Finding {
	if ctx.Sources == nil {
		return nil
	}
	var findings []Finding
	for _, fc := range ctx.ChangedFiles {
		// Changed files arrive from git as whole paths, with no walk to prune them.
		if !eligible(fc) || ctx.Skips.SkipsPath(fc.Path) {
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
