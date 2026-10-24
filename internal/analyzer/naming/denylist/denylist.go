package denylist

import (
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

const (
	ID    = "forbidden-token"
	dirID = ID + "/dir"
)

const renameHint = "rename it so no word of it is a forbidden token"

const (
	paramTokens       = "tokens"
	paramAllowedNames = "allowed_names"
)

type Analyzer struct {
	reporter    finding.Reporter
	dirReporter finding.Reporter
}

func NewAnalyzer() *Analyzer {
	return &Analyzer{
		reporter:    finding.NewReporter(ID, finding.SeverityWarning, finding.CatAST),
		dirReporter: finding.NewReporter(dirID, finding.SeverityWarning, finding.CatAST),
	}
}

type settings struct {
	reporter    finding.Reporter
	dirReporter finding.Reporter
	words       matcher
}

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []finding.Finding {
	params := ctx.Params(ID)
	tokens := params.Strings(paramTokens, nil)
	if len(tokens) == 0 {
		return nil
	}
	cfg := settings{
		reporter:    analyzer.reporter,
		dirReporter: analyzer.dirReporter,
		words:       newMatcher(tokens, params.Strings(paramAllowedNames, nil)),
	}
	findings := checkPaths(ctx, cfg)
	return append(findings, ctx.EachSyntaxFile(
		func(fc pass.FileChange, file *syntax.File) []finding.Finding {
			if fc.Generated {
				return nil
			}
			return checkNames(fc, file, cfg)
		})...)
}

func checkNames(fc pass.FileChange, file *syntax.File, cfg settings) []finding.Finding {
	var findings []finding.Finding
	for _, name := range file.Names {
		word, ok := cfg.words.hit(name.Name)
		if name.Exempt || !ok {
			continue
		}
		findings = append(findings, cfg.reporter.At(
			fc.Path,
			name.Line,
			fmt.Sprintf("%s %q contains forbidden word %q", name.Kind, name.Name, word),
			renameHint,
		))
	}
	return findings
}

func Describe() pass.Descriptor {
	return pass.Descriptor{
		ID:     ID,
		Rule:   "naming",
		Params: []string{paramTokens, paramAllowedNames},
	}
}
