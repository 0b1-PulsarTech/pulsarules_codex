package config

import (
	"reflect"
	"slices"
	"strings"
	"testing"
)

type schemaCase struct {
	name    string
	content string
	want    []string
}

func unknownKeyCases() []schemaCase {
	const head = ".pulsarules.yaml: line "
	return []schemaCase{
		{"top level", "version: 1\nbogus: 1\n", []string{head + "2: bogus: unknown key"}},
		{
			"analyzer entry",
			"version: 1\nanalyzers:\n  file-size: {bogus: 1}\n",
			[]string{head + "3: analyzers.file-size.bogus: unknown key"},
		},
		{
			"override rules",
			"version: 1\noverrides:\n  - files: [a]\n    analyzers: {file-size: {sevrity: x}}\n",
			[]string{head + "4: overrides[0].analyzers.file-size.sevrity: unknown key"},
		},
		{
			"override entry",
			"version: 1\noverrides:\n  - files: [a]\n  - flies: [b]\n",
			[]string{head + "4: overrides[1].flies: unknown key"},
		},
		{
			"delegated entry",
			"version: 1\ndelegated:\n  - {id: a}\n  - {id: b, cmd: c}\n",
			[]string{head + "4: delegated[1].cmd: unknown key"},
		},
		{
			"collects every key",
			"version: 1\nbogus: 1\nanalyzers:\n  file-size: {bogus: 1}\n",
			[]string{
				head + "2: bogus: unknown key",
				head + "4: analyzers.file-size.bogus: unknown key",
			},
		},
		{"params stay free-form", "version: 1\nanalyzers:\n  a: {params: {x: 1}}\n", nil},
	}
}

func keyShapeCases() []schemaCase {
	const head = ".pulsarules.yaml: line "
	return []schemaCase{
		{
			"complex key",
			"version: 1\n? [a, b]\n: 1\n",
			[]string{head + "2: <complex key>: unknown key"},
		},
		{
			"complex map key",
			"version: 1\nanalyzers:\n  ? [a]\n  : {}\n",
			[]string{head + "3: analyzers.<complex key>: want a string key"},
		},
		{
			"complex param key",
			"version: 1\nanalyzers:\n  big-comment:\n    params:\n      ? [a]\n      : 1\n",
			[]string{head + "5: analyzers.big-comment.params.<complex key>: want a string key"},
		},
		{
			"complex override key",
			"version: 1\noverrides:\n  - files: [a]\n    analyzers:\n      ? {x: 1}\n      : {}\n",
			[]string{head + "5: overrides[0].analyzers.<complex key>: want a string key"},
		},
		{"empty key", "version: 1\n\"\": 1\n", []string{head + "2: <empty key>: unknown key"}},
		{
			"duplicate key",
			"version: 1\nversion: 1\n",
			[]string{head + "2: version: duplicate key, first at line 1"},
		},
		{
			"alias reports the path of each use",
			"version: 1\nanalyzers:\n  file-size: &b {bogus: 1}\n  big-comment: *b\n",
			[]string{
				head + "3: analyzers.file-size.bogus: unknown key",
				head + "3: analyzers.big-comment.bogus: unknown key",
			},
		},
	}
}

func aliasLineCases() []schemaCase {
	const head = ".pulsarules.yaml: line "
	return []schemaCase{
		{
			"a kind error is pinned to the use site",
			"version: 1\nanalyzers:\n  a: &x 5\n  b: *x\n",
			[]string{
				head + "3: analyzers.a: want a mapping",
				head + "4: analyzers.b: want a mapping",
			},
		},
	}
}

func kindCases() []schemaCase {
	const head = ".pulsarules.yaml: line "
	return []schemaCase{
		{
			"scalar entry",
			"version: 1\nanalyzers:\n  file-size: off\n",
			[]string{head + "3: analyzers.file-size: want a mapping"},
		},
		{
			"overrides as a mapping",
			"version: 1\noverrides: {a: b}\n",
			[]string{head + "2: overrides: want a list"},
		},
		{
			"analyzers as a list",
			"version: 1\nanalyzers: [a]\n",
			[]string{head + "2: analyzers: want a mapping"},
		},
		{
			"override analyzers as a list",
			"version: 1\noverrides:\n  - files: [a]\n    analyzers: [file-size]\n",
			[]string{head + "4: overrides[0].analyzers: want a mapping"},
		},
	}
}

func scalarKindCases() []schemaCase {
	const head = ".pulsarules.yaml: line "
	return []schemaCase{
		{"scalar root", "5\n", []string{head + "1: want a mapping"}},
		{"list root", "[1, 2]\n", []string{head + "1: want a mapping"}},
		{
			"params as a list",
			"version: 1\nanalyzers:\n  big-comment: {params: [1]}\n",
			[]string{head + "3: analyzers.big-comment.params: want a mapping"},
		},
		{
			"tests as a scalar",
			"version: 1\ntests: web/test/**\n",
			[]string{head + "2: tests: want a list of strings"},
		},
		{
			"list item as a list",
			"version: 1\ntests: [[a]]\n",
			[]string{head + "2: tests[0]: want a string"},
		},
		{"version as a float", "version: 1.5\n", []string{head + "1: version: want an integer"}},
		{"version quoted", "version: \"1\"\n", []string{head + "1: version: want an integer"}},
		{"version in hex", "version: 0x1\n", nil},
		{"version as text", "version: abc\n", []string{head + "1: version: want an integer"}},
		{
			"run as a scalar",
			"version: 1\ndelegated:\n  - {id: x, run: a, format: dart-analyze-json}\n",
			[]string{head + "3: delegated[0].run: want a list of strings"},
		},
		{
			"enabled as text",
			"version: 1\nanalyzers:\n  file-size:\n    enabled: maybe\n",
			[]string{head + "4: analyzers.file-size.enabled: want a boolean"},
		},
		{
			"preset as a mapping",
			"version: 1\npreset: {a: 1}\n",
			[]string{head + "2: preset: want a string"},
		},
		{"null values are empty", "version: 1\nanalyzers: ~\noverrides: ~\n", nil},
	}
}

func nullCases() []schemaCase {
	const head = ".pulsarules.yaml: line "
	return []schemaCase{
		{
			"null list item",
			"version: 1\ntests: [~]\n",
			[]string{head + "2: tests[0]: want a string"},
		},
		{
			"null skip dir",
			"version: 1\nskip_dirs: [a, ~]\n",
			[]string{head + "2: skip_dirs[1]: want a string"},
		},
		{
			"null override",
			"version: 1\noverrides: [~]\n",
			[]string{head + "2: overrides[0]: want a mapping"},
		},
		{
			"null delegated entry",
			"version: 1\ndelegated: [~]\n",
			[]string{head + "2: delegated[0]: want a mapping"},
		},
		{
			"null run item",
			"version: 1\ndelegated:\n  - {id: x, run: [~], format: dart-analyze-json}\n",
			[]string{head + "3: delegated[0].run[0]: want a string"},
		},
		{
			"null analyzer entry",
			"version: 1\nanalyzers: {big-comment: ~}\n",
			[]string{head + "2: analyzers.big-comment: want a mapping"},
		},
		{"capitalised null version", "version: Null\n", nil},
		{"upper case null version", "version: NULL\n", nil},
		{
			"null override analyzer entry",
			"version: 1\noverrides:\n  - files: [a]\n    analyzers: {file-size: ~}\n",
			[]string{head + "4: overrides[0].analyzers.file-size: want a mapping"},
		},
	}
}

func nullKeyCases() []schemaCase {
	const head = ".pulsarules.yaml: line "
	return []schemaCase{
		{
			"null analyzer key",
			"version: 1\nanalyzers:\n  ~: {}\n",
			[]string{head + "3: analyzers.<null key>: want a string key"},
		},
		{
			"null param key",
			"version: 1\nanalyzers:\n  big-comment:\n    params: {~: 1}\n",
			[]string{head + "4: analyzers.big-comment.params.<null key>: want a string key"},
		},
		{"null param value", "version: 1\nanalyzers: {a: {params: {x: ~}}}\n", nil},
		{
			"null field values",
			"version: 1\npreset: ~\ntests: ~\nanalyzers: {a: {enabled: ~}}\n",
			nil,
		},
	}
}

func mergeKeySpellingCases() []schemaCase {
	const head = ".pulsarules.yaml: line "
	return []schemaCase{
		{
			"a quoted merge key is an ordinary key",
			"version: 1\n\"<<\": {}\n",
			[]string{head + "2: <<: unknown key"},
		},
		{
			"a str tagged merge key is an ordinary key",
			"version: 1\n!!str <<: {}\n",
			[]string{head + "2: <<: unknown key"},
		},
	}
}

func mergeCases() []schemaCase {
	const head = ".pulsarules.yaml: line "
	return []schemaCase{
		{
			"merge into an entry",
			"version: 1\nanalyzers:\n  file-size: &b {severity: info}\n" +
				"  big-comment:\n    <<: *b\n    enabled: true\n",
			nil,
		},
		{
			"merge into a mapping of entries",
			"version: 1\noverrides:\n  - files: [a/**]\n    analyzers: &off\n" +
				"      file-size: {enabled: false}\n  - files: [b/**]\n    analyzers:\n" +
				"      <<: *off\n      big-comment: {severity: info}\n",
			nil,
		},
		{
			"merge of a list of aliases",
			"version: 1\nanalyzers:\n  a: &x {severity: info}\n  b: &y {bogus: 1}\n" +
				"  c: {<<: [*x, *y]}\n",
			[]string{
				head + "4: analyzers.b.bogus: unknown key",
				head + "4: analyzers.c.bogus: unknown key",
			},
		},
		{
			"merge list repeating an alias",
			"version: 1\nanalyzers:\n  a: &x {bogus: 1}\n  c: {<<: [*x, *x]}\n",
			[]string{
				head + "3: analyzers.a.bogus: unknown key",
				head + "3: analyzers.c.bogus: unknown key",
			},
		},
		{
			"merged unknown key",
			"version: 1\nanalyzers:\n  a: &x {bogus: 1}\n  c: {<<: *x}\n",
			[]string{
				head + "3: analyzers.a.bogus: unknown key",
				head + "3: analyzers.c.bogus: unknown key",
			},
		},
	}
}

func assertPlainMessage(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	for _, goText := range []string{"config.", "interface", "map["} {
		if strings.Contains(err.Error(), goText) {
			t.Errorf("error %q names a Go type (%q)", err, goText)
		}
	}
}

func clipCases() []schemaCase {
	const head = ".pulsarules.yaml: line "
	return []schemaCase{
		{
			"a walker path longer than the clip keeps its tail",
			"version: 1\nanalyzers:\n  ? " + strings.Repeat("K", 1000) + "\n  : {bogus: 1}\n",
			[]string{head + "4: ..." + strings.Repeat("K", 74) + ".bogus: unknown key"},
		},
	}
}

func TestSchemaWalk(t *testing.T) {
	t.Parallel()

	for _, testCase := range slices.Concat(
		unknownKeyCases(), keyShapeCases(), aliasLineCases(), kindCases(), scalarKindCases(),
		nullCases(), nullKeyCases(), mergeCases(), mergeKeySpellingCases(), clipCases(),
	) {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			_, err := decodeFile([]byte(testCase.content))
			assertPlainMessage(t, err)
			got := ""
			if err != nil {
				got = err.Error()
			}
			if want := strings.Join(testCase.want, "; "); got != want {
				t.Fatalf("decodeFile() error = %q, want %q", got, want)
			}
		})
	}
}

func schemaStructs(typ reflect.Type, seen map[reflect.Type]bool) {
	for typ.Kind() == reflect.Pointer || typ.Kind() == reflect.Slice || typ.Kind() == reflect.Map {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct || seen[typ] {
		return
	}
	seen[typ] = true
	for field := range typ.Fields() {
		schemaStructs(field.Type, seen)
	}
}

func TestSchemaFieldsHaveExplicitTags(t *testing.T) {
	t.Parallel()

	seen := map[reflect.Type]bool{}
	schemaStructs(reflect.TypeFor[ProjectFile](), seen)
	if len(seen) < 4 {
		t.Fatalf("visited %d struct types, want the whole schema", len(seen))
	}
	for typ := range seen {
		for field := range typ.Fields() {
			tag, hasTag := field.Tag.Lookup("yaml")
			name, options, _ := strings.Cut(tag, ",")
			if !hasTag || name == "" || strings.Contains(options, "inline") {
				t.Errorf("%s.%s needs an explicit, non-inline yaml tag", typ.Name(), field.Name)
			}
		}
	}
}
