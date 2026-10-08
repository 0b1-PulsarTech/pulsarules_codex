package config

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/printable"
)

func decoderTextCases() []parseCase {
	const head = ".pulsarules.yaml: line "
	notInt := []string{head + "1: version: want an integer"}
	invalid := []string{head + "4: analyzers.big-comment.params.max_lines: invalid value"}
	params := func(value string) string {
		return "version: 1\nanalyzers:\n  big-comment:\n    params: {max_lines: " + value + "}\n"
	}
	return []parseCase{
		{"int tag on a bool", "version: !!int true\n", notInt},
		{"int tag on text", "version: !!int abc\n", notInt},
		{
			"float tag on a string field",
			"version: 1\npreset: !!float abc\n",
			[]string{head + "2: preset: want a string"},
		},
		{
			"timestamp tag on a string field",
			"version: 1\npreset: !!timestamp x\n",
			[]string{head + "2: preset: want a string"},
		},
		{"binary version", "version: !!binary \"@@\"\n", notInt},
		{"param int tag", params("!!int abc"), invalid},
		{"param float tag", params("!!float abc"), invalid},
		{"param timestamp tag", params("!!timestamp x"), invalid},
		{"param binary", params(`!!binary "@@"`), invalid},
		{
			"unknown anchor with a very long name",
			"version: 1\nx: *" + strings.Repeat("a", 250_000) + "\n",
			[]string{".pulsarules.yaml: unknown anchor '" + strings.Repeat("a", 64) + "..."},
		},
	}
}

func decoderStructureCases() []parseCase {
	const head = ".pulsarules.yaml: line "
	return []parseCase{
		{
			"binary top-level key",
			"version: 1\n? !!binary \"@@\"\n: 1\n",
			[]string{head + "2: @@: invalid key"},
		},
		{
			"binary analyzer key",
			"version: 1\nanalyzers: {!!binary \"@@\": {}}\n",
			[]string{head + "2: analyzers.@@: invalid key"},
		},
		{
			"binary param key",
			"version: 1\nanalyzers: {big-comment: {params: {!!binary \"@@\": 1}}}\n",
			[]string{head + "2: analyzers.big-comment.params.@@: invalid key"},
		},
		{
			"merge of a scalar",
			"version: 1\nanalyzers: {<<: 1}\n",
			[]string{head + "2: analyzers.<<: want a mapping or a list of mappings"},
		},
		{
			"merge of an aliased scalar",
			"version: 1\npreset: &p recommended\nanalyzers: {<<: *p}\n",
			[]string{head + "3: analyzers.<<: want a mapping or a list of mappings"},
		},
		{
			"merge of nested lists",
			"version: 1\nanalyzers:\n  file-size: {<<: [[[{enabled: true}]]]}\n",
			[]string{head + "3: analyzers.file-size.<<: want a mapping or a list of mappings"},
		},
		{
			"merge fan-out under the node budget",
			mergeBomb(4),
			[]string{head + "4: aliases expand into too many nodes"},
		},
	}
}

func nullTagCases() []parseCase {
	const head = ".pulsarules.yaml: line "
	return []parseCase{
		{
			"null tag on an integer",
			"version: !!null 5\n",
			[]string{head + "1: version: want an integer"},
		},
		{
			"null tag on a mapping field",
			"version: 1\nanalyzers: !!null x\n",
			[]string{head + "2: analyzers: want a mapping"},
		},
		{
			"null tag on a string field",
			"version: 1\npreset: !!null strict\n",
			[]string{head + "2: preset: want a string"},
		},
		{
			"null tag on a boolean field",
			"version: 1\ninclude_generated: !!null true\n",
			[]string{head + "2: include_generated: want a boolean"},
		},
		{
			"null tag on an analyzer boolean",
			"version: 1\nanalyzers: {big-comment: {enabled: !!null yes}}\n",
			[]string{head + "2: analyzers.big-comment.enabled: want a boolean"},
		},
		{
			"null tag on an override severity",
			"version: 1\noverrides:\n  - files: [a]\n" +
				"    analyzers: {file-size: {severity: !!null info}}\n",
			[]string{head + "4: overrides[0].analyzers.file-size.severity: want a string"},
		},
	}
}

func nullTagShapeCases() []parseCase {
	const head = ".pulsarules.yaml: line "
	return []parseCase{
		{
			"null tag on a mapping node",
			"version: 1\nanalyzers:\n  file-size: !!null\n    enabled: false\n",
			[]string{head + "3: analyzers.file-size: want a mapping"},
		},
		{
			"null tag on a top-level key",
			"!!null version: 1\n",
			[]string{head + "1: version: invalid key"},
		},
		{
			"null tag on an analyzer key",
			"version: 1\nanalyzers: {!!null file-size: {enabled: false}}\n",
			[]string{head + "2: analyzers.file-size: invalid key"},
		},
	}
}

func mergeTagCases() []parseCase {
	const head = ".pulsarules.yaml: line "
	const merge = ": want a mapping or a list of mappings"
	return []parseCase{
		{
			"merge tag on another key holding a mapping",
			"version: 1\nanalyzers: &m {}\n!!merge tests: *m\n",
			[]string{head + "3: tests: want a list of strings"},
		},
		{
			"merge tag on a repeated key",
			"version: 1\nanalyzers: &m {}\n!!merge version: *m\n",
			[]string{head + "3: version: duplicate key, first at line 1"},
		},
		{
			"merge tag on a string field",
			"version: 1\n!!merge preset: {}\n",
			[]string{head + "2: preset: want a string"},
		},
		{
			"merge tag on a delegated field",
			"version: 1\ndelegated: [{id: x, run: [a], format: dart-analyze-json, " +
				"!!merge dir: {}}]\n",
			[]string{head + "2: delegated[0].dir: want a string"},
		},
		{
			"merge of an aliased list in an analyzer",
			"version: 1\nanalyzers:\n  file-size: {params: {max_lines: 3}}\n" +
				"  big-comment: {params: {x: &s [{enabled: true}]}}\n  doc-comment: {<<: *s}\n",
			[]string{head + "5: analyzers.doc-comment.<<" + merge},
		},
		{
			"merge of an aliased list in params",
			"version: 1\nanalyzers:\n  file-size: {params: {max_lines: &s [{a: 1}]}}\n" +
				"  big-comment:\n    params: {<<: *s}\n",
			[]string{head + "5: analyzers.big-comment.params.<<" + merge},
		},
		{
			"merge of an aliased list beside a mistyped item",
			"version: 1\nskip_dirs: &s [{enabled: true}]\nanalyzers:\n  file-size: {<<: *s}\n",
			[]string{
				head + "2: skip_dirs[0]: want a string",
				head + "4: analyzers.file-size.<<" + merge,
			},
		},
	}
}

func TestDecodeHidesDecoderText(t *testing.T) {
	t.Parallel()

	allCases := slices.Concat(
		decoderTextCases(), decoderStructureCases(), nullTagCases(), nullTagShapeCases(),
		mergeTagCases(),
	)
	for _, testCase := range allCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			_, err := decodeFile([]byte(testCase.content))
			if err == nil || err.Error() != strings.Join(testCase.want, "; ") {
				t.Fatalf("decodeFile() error = %v, want %q", err, testCase.want)
			}
			text := strings.ReplaceAll(err.Error(), Name, "")
			for _, leaked := range []string{"cannot", "!!", "yaml:"} {
				if strings.Contains(text, leaked) {
					t.Errorf("decodeFile() error %q leaks %q", err, leaked)
				}
			}
		})
	}
}

type leftoverCase struct {
	name string
	err  error
	want string
}

func leftoverCases() []leftoverCase {
	return []leftoverCase{
		{"plain text stays", errors.New("yaml: something odd"), ".pulsarules.yaml: something odd"},
		{
			"cannot with a line",
			errors.New("yaml: line 7: cannot unmarshal !!map into string"),
			".pulsarules.yaml: line 7: invalid value",
		},
		{
			"cannot without a line",
			errors.New("yaml: cannot decode !!int `5` as a !!null"),
			".pulsarules.yaml: invalid value",
		},
		{
			"a Go type name",
			errors.New("yaml: line 2: field x not found in type config.ProjectFile"),
			".pulsarules.yaml: line 2: invalid value",
		},
		{
			"a bracketed Go type",
			errors.New("yaml: line 3: value of type []string"),
			".pulsarules.yaml: line 3: invalid value",
		},
		{
			"a map bracket alone",
			errors.New("yaml: line 4: odd map[ text"),
			".pulsarules.yaml: line 4: invalid value",
		},
		{
			"a yaml prefix inside the text alone",
			errors.New("yaml: line 5: see yaml: docs"),
			".pulsarules.yaml: line 5: invalid value",
		},
		{
			"a double bang alone",
			errors.New("yaml: line 6: odd !! text"),
			".pulsarules.yaml: line 6: invalid value",
		},
		{
			"a line prefix that is not a number",
			errors.New("yaml: line x: cannot do it"),
			".pulsarules.yaml: invalid value",
		},
	}
}

func TestLeftover(t *testing.T) {
	t.Parallel()

	for _, testCase := range leftoverCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := leftover(Name, &yaml.Node{}, testCase.err)
			if got.Error() != testCase.want {
				t.Fatalf("leftover() = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestPlainText(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		err  error
		want string
	}{
		{"prefix", errors.New("yaml: bad thing"), "bad thing"},
		{"lines", errors.New("yaml: first\n  second\nthird"), "first; second; third"},
		{
			"type errors",
			&yaml.TypeError{Errors: []string{"line 3: cannot unmarshal !!seq", "line 4: x"}},
			"line 3: cannot unmarshal !!seq; line 4: x",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := plainText(testCase.err); got != testCase.want {
				t.Fatalf("plainText() = %q, want %q", got, testCase.want)
			}
		})
	}
}

type loudValue struct {
	calls *int
}

func (value loudValue) String() string {
	*value.calls++
	return "loud"
}

func TestCollectorCapsStoredProblems(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		count      int
		wantStored int
		wantParts  int
		wantTail   string
	}{
		{"none", 0, 0, 0, ""},
		{"under the cap", 3, 3, 3, ""},
		{"exactly the cap", maxReported, maxReported, maxReported, ""},
		{"one over", maxReported + 1, maxReported, maxReported + 1, "... and 1 more problem"},
		{"many over", 5000, maxReported, maxReported + 1, "... and 4980 more problems"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			col := &Problems{path: "p.yaml"}
			for idx := range testCase.count {
				col.add(idx+1, newRoot().key("a"), "bad %d", idx)
			}
			if len(col.stored) != testCase.wantStored || col.total != testCase.count {
				t.Fatalf("stored %d of %d, want %d of %d",
					len(col.stored), col.total, testCase.wantStored, testCase.count)
			}
			err := col.err()
			if testCase.count == 0 {
				if err != nil {
					t.Fatalf("err() = %v, want nil", err)
				}
				return
			}
			parts := strings.Split(err.Error(), "; ")
			if len(parts) != testCase.wantParts {
				t.Fatalf("err() has %d parts, want %d", len(parts), testCase.wantParts)
			}
			if testCase.wantTail != "" && parts[len(parts)-1] != testCase.wantTail {
				t.Fatalf("tail = %q, want %q", parts[len(parts)-1], testCase.wantTail)
			}
		})
	}
}

func TestCollectorFormatsOnlyStoredProblems(t *testing.T) {
	t.Parallel()

	calls := 0
	col := &Problems{path: "p.yaml"}
	for range maxReported + 500 {
		col.add(1, newRoot(), "bad %v", loudValue{calls: &calls})
	}
	if calls != maxReported {
		t.Fatalf("formatted %d problems, want only the %d stored", calls, maxReported)
	}
}

func TestCollectorTextShape(t *testing.T) {
	t.Parallel()

	col := &Problems{path: "p.yaml"}
	col.add(7, newRoot().key("a").item(1), "bad")
	col.add(0, newRoot().key("b"), "no line")
	col.add(3, newRoot(), "root")
	want := []string{"p.yaml: line 7: a[1]: bad", "p.yaml: b: no line", "p.yaml: line 3: root"}
	if !slices.Equal(col.stored, want) {
		t.Fatalf("stored = %q, want %q", col.stored, want)
	}
}

func TestCollectorCapsBytes(t *testing.T) {
	t.Parallel()

	const count = 100
	col := &Problems{path: ".pulsarules.yaml"}
	for range count {
		col.add(1, newRoot().key("a"), "%s", strings.Repeat("m", 300))
	}
	if col.total != count || len(col.stored) >= maxReported || col.bytes >= maxReportedBytes+400 {
		t.Fatalf("stored %d of %d, want fewer than %d", len(col.stored), col.total, maxReported)
	}
	err := col.err()
	if err == nil || len(err.Error()) >= 4096 {
		t.Fatalf("err() = %d bytes, want under 4096", len(err.Error()))
	}
	parts := strings.Split(err.Error(), "; ")
	want := fmt.Sprintf("... and %d more problems", count-len(col.stored))
	if parts[len(parts)-1] != want {
		t.Fatalf("tail = %q, want %q", parts[len(parts)-1], want)
	}
}

func parseText(text string, _ int) string { return clipParseText(text) }

func TestClip(t *testing.T) {
	t.Parallel()

	long := strings.Repeat("x", 100)
	const bs, acc, ctl = `\`, "\u00e9", "\x01"
	testCases := []struct {
		name  string
		clip  func(string, int) string
		text  string
		limit int
		want  string
	}{
		{"tail fits", printable.Clip, "abc", 3, "abc"},
		{"tail cut", printable.Clip, "abcdefg", 3, "...efg"},
		{"tail never splits a rune", printable.Clip, "ab" + acc + acc + "x", 3, "..." + acc + "x"},
		{"tail escapes the kept part", printable.Clip, "a" + ctl + ctl + ctl, 8, `...\x01\x01`},
		{
			"tail literal backslash",
			printable.Clip,
			"xxxx" + bs + acc + "ab",
			5,
			"..." + bs + acc + "ab",
		},
		{"parse short", parseText, "line 3: bad", 0, "line 3: bad"},
		{"parse line", parseText, "line 1: " + long, 0, "line 1: " + long[:80] + "..."},
		{"parse word", parseText, "line x: " + long, 0, "line x: " + long[:72] + "..."},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := testCase.clip(testCase.text, testCase.limit); got != testCase.want {
				t.Fatalf("got %q, want %q", got, testCase.want)
			}
		})
	}
}
