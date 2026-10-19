package install

import (
	"fmt"
	"io"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/version"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/printable"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func applySelectedLayout(
	opts *version.Options,
	idx *knowledge.Index,
	prompter *Prompter,
) error {
	layout := opts.Layout
	if opts.Interactive && layout == "" {
		layout = promptLayout(idx, prompter)
	}
	if layout == "" {
		return nil
	}
	if err := idx.ApplyProfiles([]string{layout}); err != nil {
		return fmt.Errorf("apply profiles: %w", err)
	}
	_, _ = fmt.Printf("applied layout profile %q\n", layout)
	return nil
}

func promptLayout(idx *knowledge.Index, prompter *Prompter) string {
	if len(idx.Profiles) == 0 {
		return ""
	}
	printProfiles(prompter.out, idx.Profiles)
	return prompter.Ask("layout> ")
}

func printProfiles(out io.Writer, profiles []knowledge.Profile) {
	_, _ = fmt.Fprintln(out, "Available layout profiles (press Enter to skip):")
	for _, profile := range profiles {
		_, _ = fmt.Fprintf(
			out,
			"  %s [%s] - %s\n",
			printable.Line(profile.ID),
			printable.Line(profile.Axis),
			printable.Line(profile.Description),
		)
	}
}
