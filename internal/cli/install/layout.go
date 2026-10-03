package install

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/version"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func applySelectedLayout(opts *version.Options, idx *knowledge.Index) error {
	layout := opts.Layout
	if opts.Interactive && layout == "" {
		layout = promptLayout(idx)
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

func promptLayout(idx *knowledge.Index) string {
	if len(idx.Profiles) == 0 {
		return ""
	}
	printProfiles(os.Stdout, idx.Profiles)
	_, _ = fmt.Print("layout> ")
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(line)
}

func printProfiles(out io.Writer, profiles []knowledge.Profile) {
	_, _ = fmt.Fprintln(out, "Available layout profiles (press Enter to skip):")
	for _, profile := range profiles {
		_, _ = fmt.Fprintf(
			out,
			"  %s [%s] - %s\n",
			profile.ID,
			profile.Axis,
			profile.Description,
		)
	}
}
