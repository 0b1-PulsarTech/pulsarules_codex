package knowledge

import (
	"fmt"
	"io/fs"
	"testing"
	"testing/fstest"
	"unicode"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/markdown"
)

func nonASCIIProblems(root fs.FS) []string {
	var problems []string
	visit := func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		content, readErr := fs.ReadFile(root, path)
		if readErr != nil {
			return readErr
		}
		src := string(content)
		code := markdown.CodeIn(".md", src)
		line := 1
		for offset, char := range src {
			if char == '\n' {
				line++
			}
			if char <= unicode.MaxASCII || (isBoxDrawing(char) && code.Certain(offset)) {
				continue
			}
			problems = append(problems, fmt.Sprintf("%s:%d: non-ASCII %q", path, line, char))
		}
		return nil
	}
	if err := fs.WalkDir(root, "standards", visit); err != nil {
		problems = append(problems, "walk: "+err.Error())
	}
	return problems
}

func isBoxDrawing(char rune) bool {
	const first, last = 0x2500, 0x257F
	return char >= first && char <= last
}

func fenced(body string) string { return "```\n" + body + "\n```\n" }

func TestNonASCIIProblems(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		files map[string]string
		want  int
	}{
		{"clean file", map[string]string{"standards/a.md": "plain\n"}, 0},
		{"em dash in prose", map[string]string{"standards/a.md": "a\nb \u2014 c\n"}, 1},
		{"no-break space", map[string]string{"standards/a.md": "a\u00a0b\n"}, 1},
		{"em dash in a fence", map[string]string{"standards/a.md": fenced("x \u2014 y")}, 1},
		{
			"em dash in an inline span",
			map[string]string{"standards/a.md": "a `x \u2014 y` b\n"},
			1,
		},
		{"non-ASCII letter", map[string]string{"standards/a.md": "caf\u00e9\n"}, 1},
		{"emoji", map[string]string{"standards/a.md": "ok \U0001F600\n"}, 1},
		{
			"box drawing in a fence",
			map[string]string{"standards/a.md": fenced("\u251c\u2500 x")},
			0,
		},
		{
			"box drawing in an inline span",
			map[string]string{"standards/a.md": "a `\u251c x` b\n"},
			0,
		},
		{
			"box drawing in an ambiguous region",
			map[string]string{"standards/a.md": "p\n\n    \u251c x\n\nq\n"},
			1,
		},
		{"box drawing in prose", map[string]string{"standards/a.md": "\u251c x\n"}, 1},
		{"em dash after a fence", map[string]string{"standards/a.md": "```\nx\n```\n\u2014\n"}, 1},
		{"missing standards dir", map[string]string{"other/a.md": "plain\n"}, 1},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root := fstest.MapFS{}
			for path, content := range testCase.files {
				root[path] = &fstest.MapFile{Data: []byte(content)}
			}
			got := nonASCIIProblems(root)
			if len(got) != testCase.want {
				t.Fatalf("got %d problems %v, want %d", len(got), got, testCase.want)
			}
		})
	}
}

func TestStandardsAreASCII(t *testing.T) {
	t.Parallel()

	for _, problem := range nonASCIIProblems(standardsFS) {
		t.Error(problem)
	}
}
