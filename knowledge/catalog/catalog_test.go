package catalog

import "testing"

func TestCatalog(t *testing.T) {
	t.Parallel()

	var store Catalog[int]
	if store.Has("a") {
		t.Error("zero catalog Has(a) = true, want false")
	}
	if _, ok := store.Get("a"); ok {
		t.Error("zero catalog Get(a) ok = true, want false")
	}

	store.Put("a", 1)
	store.Put("a", 2)
	store.Put("b", 3)
	testCases := []struct {
		name   string
		id     string
		want   int
		wantOK bool
	}{
		{"replaced entry keeps the latest value", "a", 2, true},
		{"second entry", "b", 3, true},
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
