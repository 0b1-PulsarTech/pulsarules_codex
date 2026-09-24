package langdetect

import (
	"slices"
	"testing"
)

func TestSet(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		languages []Language
		wantHas   Language
		wantNoHas Language
		want      []string
	}{
		{name: "empty set", languages: nil, wantNoHas: Go, want: nil},
		{
			name:      "single language",
			languages: []Language{Go},
			wantHas:   Go,
			wantNoHas: Dart,
			want:      []string{"go"},
		},
		{
			name:      "multiple languages sorted regardless of input order",
			languages: []Language{TypeScript, Dart, Svelte},
			wantHas:   Svelte,
			wantNoHas: Go,
			want:      []string{"dart", "svelte", "typescript"},
		},
		{
			name:      "duplicates are deduplicated",
			languages: []Language{Go, Go, Go},
			wantHas:   Go,
			wantNoHas: Rust,
			want:      []string{"go"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			set := NewSet(testCase.languages...)
			if testCase.wantHas != "" && !set.Has(testCase.wantHas) {
				t.Errorf("Has(%q) = false, want true", testCase.wantHas)
			}
			if set.Has(testCase.wantNoHas) {
				t.Errorf("Has(%q) = true, want false", testCase.wantNoHas)
			}
			if got := set.Strings(); !slices.Equal(got, testCase.want) {
				t.Errorf("Strings() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestSetEmpty(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		set  Set
		want bool
	}{
		{name: "zero value", set: Set{}, want: true},
		{name: "constructed with nothing", set: NewSet(), want: true},
		{name: "one language", set: NewSet(Go), want: false},
		{name: "several languages", set: NewSet(Dart, Svelte), want: false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := testCase.set.Empty(); got != testCase.want {
				t.Fatalf("Empty() = %v, want %v", got, testCase.want)
			}
		})
	}
}
