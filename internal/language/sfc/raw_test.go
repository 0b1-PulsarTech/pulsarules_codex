package sfc

import "testing"

func TestCloseTag(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		rest string
		want int
	}{
		{"bracket", "a</script>", 1},
		{"blank", "a</script >", 1},
		{"slash", "a</script/", 1},
		{"newline", "a</script\n>", 1},
		{"end of text", "a</script", 1},
		{"longer name", "a</scripts>", -1},
		{"longer name then the real one", "a</scripts>b</script>", 12},
		{"absent", "abc", -1},
		{"other element", "a</style>", -1},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := closeTag(testCase.rest, "script"); got != testCase.want {
				t.Fatalf("closeTag(%q) = %d, want %d", testCase.rest, got, testCase.want)
			}
		})
	}
}
