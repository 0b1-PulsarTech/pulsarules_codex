package tsproc

import (
	"strings"
	"testing"
)

func TestTail(t *testing.T) {
	t.Parallel()

	long := strings.Repeat("a", stderrTail) + "tail"
	testCases := []struct {
		name   string
		writes []string
		want   string
	}{
		{name: "empty"},
		{name: "short", writes: []string{"boom\n"}, want: `"boom\n"`},
		{name: "joined", writes: []string{"ab", "cd"}, want: `"abcd"`},
		{name: "escaped", writes: []string{"\xff\xe2\x80\xa8"}, want: `"\xff\u2028"`},
		{
			name:   "keeps the last bytes",
			writes: []string{long},
			want:   `"` + strings.Repeat("a", stderrTail-len("tail")) + `tail"`,
		},
		{
			name:   "keeps the last bytes across writes",
			writes: []string{strings.Repeat("a", stderrTail), "bcd", "efg"},
			want:   `"` + strings.Repeat("a", stderrTail-len("bcdefg")) + `bcdefg"`,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			buffer := &tail{}
			for _, chunk := range testCase.writes {
				if wrote, err := buffer.Write([]byte(chunk)); err != nil || wrote != len(chunk) {
					t.Fatalf("write %d, %v", wrote, err)
				}
			}
			if got := buffer.String(); got != testCase.want {
				t.Fatalf("tail %q, want %q", got, testCase.want)
			}
			if len(buffer.recent) > stderrTail {
				t.Fatalf("kept %d bytes", len(buffer.recent))
			}
		})
	}
}
