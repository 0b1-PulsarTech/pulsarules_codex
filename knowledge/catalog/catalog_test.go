package catalog

import (
	"slices"
	"testing"
)

func TestCatalog_Add(t *testing.T) {
	t.Parallel()

	var store Catalog[int]
	testCases := []struct {
		name string
		id   string
		item int
		want bool
	}{
		{"first entry", "a", 1, true},
		{"second entry", "b", 3, true},
		{"duplicate id is rejected", "a", 2, false},
	}
	for _, testCase := range testCases {
		if got := store.Add(testCase.id, testCase.item); got != testCase.want {
			t.Errorf("%s: Add(%q) = %v, want %v",
				testCase.name, testCase.id, got, testCase.want)
		}
	}
	if got, _ := store.Get("a"); got != 1 {
		t.Errorf("duplicate Add replaced the entry: Get(a) = %d, want 1", got)
	}
	if got := slices.Collect(store.All()); !slices.Equal(got, []int{1, 3}) {
		t.Errorf("All = %v, want [1 3]", got)
	}
}

func TestCatalog_Lookup(t *testing.T) {
	t.Parallel()

	var store Catalog[int]
	if store.Has("a") {
		t.Error("zero catalog Has(a) = true, want false")
	}
	store.Replace("a", 9)
	if store.Has("a") {
		t.Error("Replace on an unknown id inserted it")
	}
	for id, item := range map[string]int{"a": 1, "b": 3} {
		if !store.Add(id, item) {
			t.Fatalf("Add(%q) = false, want true", id)
		}
	}
	store.Replace("a", 2)
	testCases := []struct {
		name   string
		id     string
		want   int
		wantOK bool
	}{
		{"replaced entry", "a", 2, true},
		{"other entry", "b", 3, true},
		{"unknown id", "c", 0, false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got, ok := store.Get(testCase.id)
			if got != testCase.want || ok != testCase.wantOK {
				t.Errorf("Get(%q) = %d, %v; want %d, %v",
					testCase.id, got, ok, testCase.want, testCase.wantOK)
			}
			if store.Has(testCase.id) != testCase.wantOK {
				t.Errorf("Has(%q) = %v, want %v", testCase.id, !testCase.wantOK, testCase.wantOK)
			}
		})
	}
}

func TestCatalog_AllStopsEarly(t *testing.T) {
	t.Parallel()

	var store Catalog[int]
	for pos, id := range []string{"a", "b", "c"} {
		if !store.Add(id, pos) {
			t.Fatalf("Add(%q) = false, want true", id)
		}
	}
	var seen []int
	for item := range store.All() {
		seen = append(seen, item)
		break
	}
	if !slices.Equal(seen, []int{0}) {
		t.Errorf("early break saw %v, want [0]", seen)
	}
}
