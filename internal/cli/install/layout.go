package install

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/cliopts"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// applySelectedLayout resolves the layout profile (prompting when interactive) and
// applies it to the index before rendering.
func applySelectedLayout(opts *cliopts.Options, idx *knowledge.Index) error {
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

// promptLayout lists the available customization profiles and reads one layout id
// from stdin. An empty line selects no profile.
func promptLayout(idx *knowledge.Index) string {
	if len(idx.Profiles) == 0 {
		return ""
	}
	_, _ = fmt.Println("Available layout profiles (press Enter to skip):")
	for _, profile := range idx.Profiles {
		_, _ = fmt.Printf("  %s [%s] - %s\n", profile.ID, profile.Axis, profile.Description)
	}
	_, _ = fmt.Print("layout> ")
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(line)
}
