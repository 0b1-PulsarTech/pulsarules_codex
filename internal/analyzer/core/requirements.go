package core

// Severity classifies how a finding should affect the hook's exit behavior.
type Severity int

const (
	// SeverityInfo is a non-blocking observation (e.g. a suggestion).
	SeverityInfo Severity = iota
	// SeverityWarning is a non-blocking admonition the agent should address.
	SeverityWarning
	// SeverityError is a blocking finding that should reject the operation.
	SeverityError
)

// Category classifies the analysis technique a finding comes from, so the
// pipeline can group and filter output.
type Category int

const (
	// CatSyntax covers pure text/structure checks (regex, BNF parsing).
	CatSyntax Category = iota
	// CatAST covers Go AST-based checks (go/parser + go/ast + go/types).
	CatAST
	// CatArch covers multi-file architecture checks (import graph, boundaries).
	CatArch
	// CatCommit covers commit-message validation.
	CatCommit
	// CatProject covers project-level config validation.
	CatProject
)

// StageID identifies an ordered stage in the pipeline. Each stage runs after
// the previous one, and its analyzers receive the accumulated context.
type StageID int

const (
	// StageContext builds the pipeline context (git state, changed files, ASTs).
	StageContext StageID = iota
	// StageStatic runs pure text/structure analyzers (commit BNF, file size).
	StageStatic
	// StageAST runs Go AST-based analyzers (else, shadowing, complexity).
	StageAST
	// StageArch runs architecture analyzers (dependency direction, boundaries).
	StageArch
	// StageRuleInjection attaches embedded rule markdown to findings.
	StageRuleInjection
	// StageOutput aggregates and formats findings for the hook output.
	StageOutput
)

// Requirements is a compatibility shim: the pipeline no longer gates an
// analyzer on it (Needs was dropped from the Analyzer interface and
// pipeline.requirementsMet was deleted - NeedsAST tested ctx.ChangedFiles
// instead of ctx.ASTCache, so a nil cache with non-nil ChangedFiles passed
// anyway, and NeedsGitHistory had zero callers). It survives only because
// internal/analyzer/delegation still declares a Needs() core.Requirements
// method this package does not own; deleting Requirements is the last step
// once delegation drops that method too.
type Requirements struct {
	// NeedsAST is unread by the pipeline; kept for delegation's Needs().
	NeedsAST bool
	// NeedsGitHistory is unread by the pipeline; kept for delegation's Needs().
	NeedsGitHistory bool
}
