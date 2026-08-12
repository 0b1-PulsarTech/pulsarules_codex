package golangcilint

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

func hasGoWork(projectDir string) bool {
	_, err := os.Stat(filepath.Join(projectDir, "go.work"))
	return err == nil
}

// goWorkUsePrefix is the go.work directive line prefix this parser
// recognizes; the module path starts right after it.
const goWorkUsePrefix = "use "

func parseGoWorkModules(projectDir string) []string {
	//nolint:gosec // G304: projectDir is the analyzed project root, not user input.
	f, err := os.Open(filepath.Join(projectDir, "go.work"))
	if err != nil {
		return nil
	}
	defer func() { _ = f.Close() }()

	var modules []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if rest, ok := strings.CutPrefix(line, goWorkUsePrefix); ok {
			mod := strings.Trim(strings.TrimSpace(rest), "\"")
			if mod != "" {
				modules = append(modules, mod)
			}
		}
	}
	return modules
}

func resolvedConfigFlag(projectDir, configPath string) string {
	if configPath != "" {
		return configPath
	}
	for _, candidate := range []string{
		projectDir,
		filepath.Join(projectDir, "tools"),
		filepath.Join(projectDir, "build"),
	} {
		path := filepath.Join(candidate, ".golangci.yml")
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

func extractJSON(out []byte) []byte {
	start := -1
	for i, b := range out {
		if b == '{' {
			start = i
			break
		}
	}
	if start < 0 {
		return out
	}
	end := -1
	for i := len(out) - 1; i >= start; i-- {
		if out[i] == '}' {
			end = i
			break
		}
	}
	if end < 0 {
		return out
	}
	return out[start : end+1]
}
