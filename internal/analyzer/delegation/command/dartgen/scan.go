package dartgen

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

func Missing(root, prefix string, files []string) []string {
	directive := regexp.MustCompile(`(?m)^\s*(?:part|import|export)\s+['"]([^'"\n]+)['"]`)
	// Directives sit at the top of a Dart file, so a bounded read finds them.
	seen := map[string]struct{}{}
	for _, file := range files {
		if !strings.HasSuffix(file, ".dart") || !strings.HasPrefix(file, prefix) {
			continue
		}
		for _, match := range directive.FindAllStringSubmatch(head(root, file), -1) {
			target := Resolve(file, match[1])
			if target == "" || !IsGenerated(target) || exists(root, target) {
				continue
			}
			seen[target] = struct{}{}
		}
	}
	missing := make([]string, 0, len(seen))
	for target := range seen {
		missing = append(missing, target)
	}
	slices.Sort(missing)
	return missing
}

func head(root, file string) string {
	const readCap = 64 << 10

	// An unreadable file only hides its own directives; the analyzer run still reports it.
	confined, err := os.OpenRoot(root)
	if err != nil {
		return ""
	}
	defer func() { _ = confined.Close() }()
	handle, err := confined.Open(file)
	if err != nil {
		return ""
	}
	defer func() { _ = handle.Close() }()
	text, err := io.ReadAll(io.LimitReader(handle, readCap))
	if err != nil {
		return ""
	}
	return string(text)
}

func exists(root, target string) bool {
	_, err := os.Stat(filepath.Join(root, filepath.FromSlash(target)))
	return !errors.Is(err, fs.ErrNotExist)
}
