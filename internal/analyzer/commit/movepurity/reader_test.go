package movepurity

import "testing"

func TestCachedReader_ReadsEachBlobOnce(t *testing.T) {
	t.Parallel()

	lookups := 0
	repo := countingRepo{
		fakeRepo: newFakeRepo(map[string]string{"a.go": "package a\n"}, map[string]string{}),
		lookups:  &lookups,
	}
	testCases := []struct {
		name string
		path string
		want error
	}{
		{name: "present blob", path: "a.go"},
		{name: "missing blob", path: "gone.go", want: errMissing},
	}
	reader := newCachedReader(repo)
	for _, testCase := range testCases {
		before := lookups
		for range 2 {
			if _, err := reader.HeadBlob(testCase.path); err != testCase.want {
				t.Fatalf("%s: HeadBlob() error = %v, want %v", testCase.name, err, testCase.want)
			}
		}
		if lookups-before != 1 {
			t.Fatalf("%s: %d reads of the underlying repo, want 1", testCase.name, lookups-before)
		}
	}
}
