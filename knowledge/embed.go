package knowledge

import "embed"

// standardsFS holds the committed rules, patterns, workflows, skills, and router
// data. With no --root flag the binary reads this snapshot; with --root it reads
// from disk for dev edits.
//
//go:embed standards
var standardsFS embed.FS

// templatesFS holds the Markdown and shell templates used to render skills and
// install hooks.
//
//go:embed templates
var templatesFS embed.FS
