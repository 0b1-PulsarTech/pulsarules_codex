package registry

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/commit/message"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/commit/movepurity"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

func Params() map[string][]string {
	// Every key an analyzer reads through core.ParamSet; params_test.go scans for misses.
	extensions := pass.ParamExtensions
	return map[string][]string{
		"file-size":           {"max_lines", extensions},
		"big-comment":         {"max_lines", extensions},
		"text-markers":        {extensions},
		"typographic-markers": {extensions},
		"top-of-file":         {extensions},
		"simplification-path": {extensions},
		"history-narration":   {extensions},
		"package-file-count":  {"max_files", "exempt", extensions},
		"complexity":          {"max_complexity", "max_func_lines", "max_params"},
		"single-use-declaration": {
			"check_const", "check_var", "pinned_annotations",
			"stateful_type_prefixes", "hoisted_init_kinds",
		},
		"single-letter-name":  {"exported_only"},
		"too-many-results":    {"max_results"},
		"body-header-comment": {"min_lines"},
		"branch-name":         {"extra_types"},
		"golangci-lint":       {"config_path"},
		message.ID: {
			message.ParamEmojiHardWindow,
			message.ParamEmojiSoftWindow,
			message.ParamEmojiSuggestions,
		},
		movepurity.ID: {movepurity.ParamMinSimilarity},
	}
}
