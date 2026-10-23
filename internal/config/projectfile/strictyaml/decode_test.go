package strictyaml

import (
	"slices"
	"strings"
	"testing"
)

func TestDecodeErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		content string
		want    string
	}{
		{"empty file", "", ".pulsarules.yaml: version: required"},
		{"comment only", "# nothing\n", ".pulsarules.yaml: version: required"},
		{
			"not yaml",
			"version: [",
			".pulsarules.yaml: line 1: did not find expected node content",
		},
		{
			"bad indent",
			"version: 1\n  bad: indent\n",
			".pulsarules.yaml: line 2: mapping values are not allowed in this context",
		},
		{
			"second document",
			"version: 1\n---\nversion: 1\n",
			".pulsarules.yaml: only one YAML document is allowed",
		},
		{
			"second document with a bare null",
			"version: 1\n---\nnull\n",
			".pulsarules.yaml: only one YAML document is allowed",
		},
		{
			"second document that is not yaml",
			"version: 1\n---\n  : bad: [\n",
			".pulsarules.yaml: only one YAML document is allowed",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			_, err := decodeFixture([]byte(testCase.content))
			if err == nil || err.Error() != testCase.want {
				t.Fatalf("decodeFixture() error = %v, want %q", err, testCase.want)
			}
			assertPlainFixture(t, err)
			if strings.Contains(err.Error(), ".yaml: yaml:") {
				t.Errorf("decodeFixture() error %q keeps the yaml: prefix", err)
			}
		})
	}
}

func TestDecodeTrailingEmptyDocuments(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		content string
	}{
		{"separator", "version: 1\n---\n"},
		{"separator and comment", "version: 1\n---\n# trailing\n"},
		{"two separators", "version: 1\n---\n---\n"},
		{"document end", "version: 1\n...\n"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			file, err := decodeFixture([]byte(testCase.content))
			if err != nil || file.Version != 1 {
				t.Fatalf("decodeFixture() = %+v, %v, want version 1", file, err)
			}
		})
	}
}

func mergePanicCases() []schemaCase {
	const head = ".pulsarules.yaml: line "
	return []schemaCase{
		{
			"alias key beside a merge",
			"version: 1\nb: &b {x: 1}\nanalyzers:\n  doc-comment: {<<: ~, *b : 1}\n",
			[]string{
				head + "2: b: unknown key",
				head + "4: analyzers.doc-comment.<<: want a mapping or a list of mappings",
				head + "4: analyzers.doc-comment.<complex key>: unknown key",
			},
		},
		{
			"collection key that holds a merge and a list key",
			"version: 1\nanalyzers:\n  a: {<<: ~, [{<<: ~, [a]: 1}]: 1}\n",
			[]string{
				head + "3: analyzers.a.<<: want a mapping or a list of mappings",
				head + "3: analyzers.a.<complex key>: unknown key",
			},
		},
		{
			"list key beside a merge at the top",
			"version: 1\n<<: ~\n[a]: 1\n",
			[]string{
				head + "2: <<: want a mapping or a list of mappings",
				head + "3: <complex key>: unknown key",
			},
		},
		{
			"list key beside a merge in an entry",
			"version: 1\nanalyzers:\n  a: {<<: ~, [a]: 1}\n",
			[]string{
				head + "3: analyzers.a.<<: want a mapping or a list of mappings",
				head + "3: analyzers.a.<complex key>: unknown key",
			},
		},
		{
			"map key beside a merge in a map",
			"version: 1\nanalyzers: {<<: ~, {a: 1}: 1}\n",
			[]string{
				head + "2: analyzers.<<: want a mapping or a list of mappings",
				head + "2: analyzers.<complex key>: want a string key",
			},
		},
		{
			"merge list of aliases beside an alias key",
			"version: 1\nm: &m {a: 1}\nk: &k {b: 2}\nanalyzers:\n  a: {<<: [*m], *k" +
				" : 1}\n",
			[]string{
				head + "2: m: unknown key",
				head + "3: k: unknown key",
				head + "2: analyzers.a.a: unknown key",
				head + "5: analyzers.a.<complex key>: unknown key",
			},
		},
		{
			"merged list key under a non-string params key",
			"version: 1\nanalyzers:\n  a:\n    params: {k: {1: x, <<: {[a]: 1}}}\n",
			[]string{
				head + "4: analyzers.a.params.k: invalid value",
			},
		},
		{
			"merged list key inside a params list item",
			"version: 1\nanalyzers:\n  a:\n    params: {k: [{1: a, <<: {[a]: 1}}]}" +
				"\n",
			[]string{
				head + "4: analyzers.a.params.k: invalid value",
			},
		},
	}
}

func keyTypePanicCases() []schemaCase {
	const head = ".pulsarules.yaml: line "
	return []schemaCase{
		{
			"merge shadowed by key type in analyzers",
			"version: 1\nanalyzers:\n  1: {}\n  <<: {1: {<<: ~, [x]: 1}}\n",
			[]string{
				".pulsarules.yaml: invalid value",
			},
		},
		{
			"binary key that spells a field name",
			"version: 1\n!!binary YW5hbHl6ZXJz:\n  doc-comment: {<<: ~, [x]: 1}\n",
			[]string{
				head + "2: YW5hbHl6ZXJz: unknown key",
			},
		},
		{
			"merge shadowed by key type in params",
			"version: 1\nanalyzers:\n  doc-comment:\n    params:\n      1: x\n     " +
				" <<: {\"1\": {1: x, <<: {[a]: 1}}}\n",
			[]string{
				".pulsarules.yaml: invalid value",
			},
		},
		{
			"tagged empty list key in an entry",
			"version: 1\nanalyzers:\n  doc-comment: {<<: ~, {!!str []: x}: 1}\n",
			[]string{
				head + "3: analyzers.doc-comment.<<: want a mapping or a list of mappings",
				head + "3: analyzers.doc-comment.<complex key>: unknown key",
			},
		},
		{
			"tagged empty list key in params",
			"version: 1\nanalyzers:\n  doc-comment:\n    params: {k: {1: x, <<: {{!" +
				"!str []: x}: 1}}}\n",
			[]string{
				head + "4: analyzers.doc-comment.params.k: invalid value",
			},
		},
		{
			"tagged empty list key at the top",
			"version: 1\n<<: ~\n{!!str []: x} : x\n",
			[]string{
				head + "2: <<: want a mapping or a list of mappings",
				head + "3: <complex key>: unknown key",
			},
		},
		{
			"null-tagged mapping holding a list key",
			"version: 1\nanalyzers:\n  doc-comment: !!null {<<: ~, [x]: 1}\n",
			[]string{
				head + "3: analyzers.doc-comment: want a mapping",
			},
		},
	}
}

func panicCases() []schemaCase {
	return slices.Concat(mergePanicCases(), keyTypePanicCases())
}

func plainKeyIdentityCases() []schemaCase {
	const head = ".pulsarules.yaml: line "
	return []schemaCase{
		{
			"keeps a list key without a merge",
			"version: 1\nanalyzers:\n  a: {[a]: 1}\n",
			[]string{
				head + "3: analyzers.a.<complex key>: unknown key",
			},
		},
		{
			"keeps a merge inside a merged params value",
			"version: 1\nanalyzers:\n  a:\n    params: {k: {<<: {<<: ~, [a]: 1}}}\n",
			[]string{
				head + "4: analyzers.a.params.k: invalid value",
			},
		},
		{
			"keeps a params alias key beside a merge",
			"version: 1\nanalyzers:\n  a:\n    params: {x: &x {a: 1}, k: {<<: ~, *x" +
				" : 1}}\n",
			[]string{
				head + "4: analyzers.a.params.k: invalid value",
			},
		},
		{
			"keeps a params list key without a merge",
			"version: 1\nanalyzers:\n  a:\n    params: {k: {[a]: 1}}\n",
			[]string{
				head + "4: analyzers.a.params.k: invalid value",
			},
		},
		{
			"keeps an unknown key holding a merge and a list key",
			"version: 1\nunknown: {<<: ~, [a]: 1}\n",
			[]string{
				head + "2: unknown: unknown key",
			},
		},
		{
			"keeps a binary key shadowing a merge source",
			"version: 1\nanalyzers: {!!binary YQ==: {}, <<: {a: {<<: ~, [x]: 1}}}\n",
			[]string{
				head + "2: analyzers.a.<<: want a mapping or a list of mappings",
				head + "2: analyzers.a.<complex key>: unknown key",
			},
		},
	}
}

func orderIdentityCases() []schemaCase {
	const head = ".pulsarules.yaml: line "
	return []schemaCase{
		{
			"keeps a binary key shadowing a merge source beside a bad preset",
			"version: 1\npreset: bogus\nanalyzers: {!!binary YQ==: {}, <<: {a: {<<:" +
				" ~, [x]: 1}}}\n",
			[]string{
				head + "3: analyzers.a.<<: want a mapping or a list of mappings",
				head + "3: analyzers.a.<complex key>: unknown key",
			},
		},
		{
			"keeps a params merge of a scalar before a hashing merge",
			"version: 1\nanalyzers:\n  doc-comment:\n    params: {k: {a: {<<: 1}, b" +
				": {1: x, <<: {[a]: 1}}}}\n",
			[]string{
				head + "4: analyzers.doc-comment.params.k: invalid value",
			},
		},
		{
			"keeps a params bad int before a hashing merge",
			"version: 1\nanalyzers:\n  doc-comment:\n    params: {k: {a: !!int x, b" +
				": {1: x, <<: {[a]: 1}}}}\n",
			[]string{
				head + "4: analyzers.doc-comment.params.k: invalid value",
			},
		},
		{
			"keeps a params bad binary before a hashing merge",
			"version: 1\nanalyzers:\n  doc-comment:\n    params: {k: {a: !!binary " +
				"\"@@@\", b: {1: x, <<: {[a]: 1}}}}\n",
			[]string{
				head + "4: analyzers.doc-comment.params.k: invalid value",
			},
		},
		{
			"keeps a params list key before a hashing merge",
			"version: 1\nanalyzers:\n  doc-comment:\n    params: {k: {a: {[q]: 1}, " +
				"b: {1: x, <<: {[a]: 1}}}}\n",
			[]string{
				head + "4: analyzers.doc-comment.params.k: invalid value",
			},
		},
		{
			"keeps a params list of merges ending before a hashing merge",
			"version: 1\nanalyzers:\n  doc-comment:\n    params: {k: [{<<: 1}, {1: " +
				"x, <<: {[a]: 1}}]}\n",
			[]string{
				head + "4: analyzers.doc-comment.params.k: invalid value",
			},
		},
	}
}

func shadowIdentityCases() []schemaCase {
	const head = ".pulsarules.yaml: line "
	return []schemaCase{
		{
			"keeps a params merge source shadowed by its owner",
			"version: 1\nanalyzers:\n  doc-comment:\n    params: {k: {a: 1, <<: {a:" +
				" {<<: ~, !!str [x]: 1}}}}\n",
			[]string{},
		},
		{
			"keeps a params merge source shadowed in a general map",
			"version: 1\nanalyzers:\n  doc-comment:\n    params: {k: {1: x, a: 1, <" +
				"<: {a: {1: y, <<: {[z]: 1}}}}}\n",
			[]string{},
		},
		{
			"keeps a params key with nested merges and tagged keys",
			"version: 1\nanalyzers:\n  doc-comment:\n    params: {k: {<<: {}, !!str" +
				" [{<<: ~, !!str [{[b]: 1}]: 1}]: 1}}\n",
			[]string{
				head + "4: analyzers.doc-comment.params.k: invalid value",
			},
		},
		{
			"keeps an entry key with nested merges and tagged keys",
			"version: 1\nanalyzers:\n  doc-comment: {<<: ~, [{<<: ~, !!str [{[b]: 1" +
				"}]: 1}]: 1}\n",
			[]string{
				head + "3: analyzers.doc-comment.<<: want a mapping or a list of mappings",
				head + "3: analyzers.doc-comment.<complex key>: unknown key",
			},
		},
		{
			"keeps a top-level key with a merge and a tagged map key",
			"version: 1\n<<: {}\n{<<: x, !!str {a: x}: b} : 1\n",
			[]string{
				head + "3: <complex key>: unknown key",
			},
		},
	}
}

func byteIdentityCases() []schemaCase {
	return slices.Concat(plainKeyIdentityCases(), orderIdentityCases(), shadowIdentityCases())
}

func TestDecodeSurvivesYamlPanics(t *testing.T) {
	t.Parallel()

	for _, testCase := range slices.Concat(panicCases(), byteIdentityCases()) {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			_, err := decodeFixture([]byte(testCase.content))
			got := ""
			if err != nil {
				got = err.Error()
			}
			if want := strings.Join(testCase.want, "; "); got != want {
				t.Fatalf("decodeFixture() error = %q, want %q", got, want)
			}
		})
	}
}
