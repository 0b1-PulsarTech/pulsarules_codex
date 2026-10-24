package denylist

import (
	"fmt"
	"path"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

func checkPaths(ctx *pass.Context, cfg settings) []finding.Finding {
	var findings []finding.Finding
	seen := map[string]bool{}
	for _, fc := range ctx.ChangedFiles {
		if fc.Generated || ctx.Skips.SkipsPath(fc.Path) || isGone(ctx, fc.Path) {
			continue
		}
		if word, ok := cfg.words.hitFile(path.Base(fc.Path)); ok {
			findings = append(findings, cfg.reporter.At(
				fc.Path, 0, fmt.Sprintf("file name contains forbidden word %q", word), renameHint,
			))
		}
		findings = append(findings, checkDirs(fc.Path, cfg, seen)...)
	}
	return findings
}

func isGone(ctx *pass.Context, file string) bool {
	if ctx.Sources == nil {
		return false
	}
	_, ok := ctx.Sources.Read(file)
	return !ok
}

func checkDirs(file string, cfg settings, seen map[string]bool) []finding.Finding {
	var findings []finding.Finding
	segments := strings.Split(path.Dir(file), "/")
	for depth, segment := range segments {
		prefix := strings.Join(segments[:depth+1], "/")
		if segment == "." || seen[prefix] {
			continue
		}
		seen[prefix] = true
		if word, ok := cfg.words.hit(segment); ok {
			message := fmt.Sprintf("directory name contains forbidden word %q", word)
			findings = append(findings, cfg.dirReporter.At(prefix, 0, message, renameHint))
		}
	}
	return findings
}
