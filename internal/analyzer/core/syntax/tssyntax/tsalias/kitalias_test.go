package tsalias

import (
	"slices"
	"strings"
	"testing"
)

func TestFinder_KitAliases(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		config string
		absent bool
		want   []Alias
	}{
		{
			name: "flat object",
			config: `export default { kit: { alias: {
				'$components': 'src/components',
				"@/*": "./src/*",
			} } };`,
			want: []Alias{
				kitAlias("web", "$components", "web/src/components"),
				kitAlias("web", "$components/*", "web/src/components/*"),
				kitAlias("web", "@/*", "web/src/*"),
			},
		},
		{
			name:   "no trailing comma and tight spacing",
			config: `kit:{alias:{'$a':'a'}}`,
			want: []Alias{
				kitAlias("web", "$a", "web/a"), kitAlias("web", "$a/*", "web/a/*"),
			},
		},
		{name: "empty object", config: `kit: { alias: {} }`},
		{name: "computed value is ignored", config: `alias: { '$a': path.resolve('a') }`},
		{name: "template literal is ignored", config: "alias: { '$a': `a` }"},
		{name: "spread is ignored", config: `alias: { ...base, '$a': 'a' }`},
		{name: "unquoted key is ignored", config: `alias: { $a: 'a' }`},
		{name: "escaped value is ignored", config: `alias: { '$a': 'a\'b' }`},
		{name: "multi line string is ignored", config: "alias: { '$a': 'a\nb' }"},
		{name: "unclosed string", config: `alias: { '$a': 'a`},
		{name: "missing colon", config: `alias: { '$a' 'a' }`},
		{name: "missing comma", config: `alias: { '$a': 'a' '$b': 'b' }`},
		{name: "not an object", config: `alias: 3`},
		{name: "unrelated word", config: `const aliases = [];`},
		{
			name:   "target outside the repo is skipped",
			config: `alias: { '$a': '../../a', '': 'b' }`,
		},
		{
			name:   "first flat occurrence after a failing one",
			config: `alias: base; kit: { alias: { '$z': 'z' } }`,
			want:   []Alias{kitAlias("web", "$z", "web/z"), kitAlias("web", "$z/*", "web/z/*")},
		},
		{name: "oversized config", config: strings.Repeat(" ", maxConfigBytes+1)},
		{name: "missing file", absent: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			files := map[string]string{}
			if !testCase.absent {
				files["web/svelte.config.js"] = testCase.config
			}
			got := NewFinder(readerOf(files)).kitAliases("web")
			if !slices.Equal(got, testCase.want) {
				t.Fatalf("aliases\n got %v\nwant %v", got, testCase.want)
			}
		})
	}
}

func TestFinder_ForReadsKitAliases(t *testing.T) {
	t.Parallel()

	files := map[string]string{
		"web/package.json":     `{"devDependencies": {"@sveltejs/kit": "^2"}}`,
		"web/svelte.config.js": `export default { kit: { alias: { '$ui': 'src/ui' } } };`,
	}
	got := NewFinder(readerOf(files)).For("web/src/a.svelte")
	want := []Alias{
		kitAlias("web", "$lib/*", "web/src/lib/*"),
		kitAlias("web", "$ui", "web/src/ui"),
		kitAlias("web", "$ui/*", "web/src/ui/*"),
	}
	if !slices.Equal(got, want) {
		t.Fatalf("aliases\n got %v\nwant %v", got, want)
	}
}
