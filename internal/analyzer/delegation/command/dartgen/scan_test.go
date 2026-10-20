package dartgen

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func writeTree(t *testing.T, files map[string]string) (string, []string) {
	t.Helper()
	root := t.TempDir()
	names := make([]string, 0, len(files))
	for name, text := range files {
		full := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o700); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(text), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
		names = append(names, name)
	}
	slices.Sort(names)
	return root, names
}

func padTo(size int, tail string) string {
	return strings.Repeat("/", size-len(tail)-1) + "\n" + tail
}

type missingCase struct {
	name   string
	files  map[string]string
	prefix string
	want   []string
}

func missingCases() []missingCase {
	return []missingCase{
		{
			name:  "export of an absent generated file",
			files: map[string]string{"lib/n.dart": "export 'src/c.g.dart';\n"},
			want:  []string{"lib/src/c.g.dart"},
		},
		{
			"part and import with double quotes",
			map[string]string{
				"lib/a.dart": "import \"a.freezed.dart\";\npart 'a.g.dart';\n",
			},
			"",
			[]string{"lib/a.freezed.dart", "lib/a.g.dart"},
		},
		{
			"present generated file",
			map[string]string{
				"lib/n.dart": "export 'c.g.dart';\n", "lib/c.g.dart": "",
			},
			"",
			[]string{},
		},
		{
			"plain, package and dart imports",
			map[string]string{
				"lib/n.dart": "import 'b.dart';\nimport 'package:x/y.g.dart';\n" +
					"import 'dart:io';\n",
			},
			"",
			[]string{},
		},
		{
			"part of is not a directive target",
			map[string]string{"lib/a.g.dart": "part of 'a.dart';\n"},
			"",
			[]string{},
		},
	}
}

func moreMissingCases() []missingCase {
	return []missingCase{
		{
			"non dart file is ignored",
			map[string]string{"lib/n.txt": "export 'c.g.dart';\n"},
			"",
			[]string{},
		},
		{
			"prefix limits the scan",
			map[string]string{
				"a/lib/n.dart": "export 'c.g.dart';\n", "b/lib/n.dart": "export 'd.g.dart';\n",
			},
			"b/",
			[]string{"b/lib/d.g.dart"},
		},
		{
			"directive past the read cap is not seen",
			map[string]string{
				"lib/n.dart": strings.Repeat("//\n", 100000) + "export 'c.g.dart';\n",
			},
			"",
			[]string{},
		},
		{
			"directive ending one byte past the read cap is not seen",
			map[string]string{"lib/n.dart": padTo(64<<10+1, "export 'c.g.dart'")},
			"",
			[]string{},
		},
		{
			"directive ending at the read cap is seen",
			map[string]string{"lib/n.dart": padTo(64<<10, "export 'c.g.dart'")},
			"",
			[]string{"lib/c.g.dart"},
		},
		{
			"same target from two files is reported once",
			map[string]string{
				"lib/a.dart": "export 'c.g.dart';\n", "lib/b.dart": "import 'c.g.dart';\n",
			},
			"",
			[]string{"lib/c.g.dart"},
		},
	}
}

func TestMissing(t *testing.T) {
	t.Parallel()

	for _, testCase := range append(missingCases(), moreMissingCases()...) {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root, names := writeTree(t, testCase.files)
			if got := Missing(root, testCase.prefix, names); !slices.Equal(got, testCase.want) {
				t.Errorf("Missing = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestMissingUnreadableListedFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if got := Missing(root, "", []string{"lib/gone.dart"}); len(got) != 0 {
		t.Errorf("Missing = %v, want none", got)
	}
}
