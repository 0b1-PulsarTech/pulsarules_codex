package projectfile

import (
	"fmt"
	"testing"
)

func TestDecodeRejectsDirectives(t *testing.T) {
	t.Parallel()

	const rejected = ".pulsarules.yaml: line %d: YAML directives are not allowed"
	testCases := []struct {
		name     string
		content  string
		wantLine int
	}{
		{"yaml directive", "%YAML 1.2\n---\nversion: 1\n", 1},
		{"tag directive after a document end", "version: 1\n...\n%TAG ! tag:a:\n---\nx: 1\n", 3},
		{"directive after a byte order mark", "\xef\xbb\xbf%YAML 1.2\n---\nversion: 1\n", 1},
		{"directive after carriage returns", "version: 1\r\n...\r\n%YAML 1.2\r\n", 3},
		{
			"directive after lone carriage returns",
			"version: 1\r...\r%TAG ! a! tag:x:\r---\r", 3,
		},
		{"directive after a comment and a carriage return", "#c\r%TAG ! a! x\r", 2},
		{"directive after a next line character", "#c\xc2\x85%TAG ! a! x\n", 2},
		{"directive after a line separator", "#c\xe2\x80\xa8%TAG ! a! x\n", 2},
		{"directive after a paragraph separator", "#c\xe2\x80\xa9%TAG ! a! x\n", 2},
		{"directive after a document end and a next line", "version: 1\n...\n\xc2\x85%TAG !\n", 4},
		{"line feed then carriage return counts twice", "#c\n\r%TAG ! a! x\n", 3},
		{"directive after a byte order mark and a break", "\xef\xbb\xbf#\r%YAML 1.2\r", 2},
		{"indented block scalar line", "version: 1\npreset: |\n  %not a directive\n", 0},
		{
			"quoted scalar continued at column 0 is a false positive kept",
			"version: 1\nskip_dirs: [\"a\n%b\"]\n", 3,
		},
		{"percent after a character ending in 0x85", "version: 1\npreset: \"\xc5\x85%x\"\n", 0},
		{"percent after a character starting with 0xe2", "version: 1\npreset: \"\u2022%x\"\n", 0},
		{"percent inside a quoted value", "version: 1\npreset: \"\\n%x\"\n", 0},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			_, err := decodeFile([]byte(testCase.content))
			if testCase.wantLine == 0 {
				if err != nil {
					t.Fatalf("decodeFile() error = %v, want nil", err)
				}
				return
			}
			want := fmt.Sprintf(rejected, testCase.wantLine)
			if err == nil || err.Error() != want {
				t.Fatalf("decodeFile() error = %v, want %q", err, want)
			}
		})
	}
}

func TestDecodeRejectsNonUTF8(t *testing.T) {
	t.Parallel()

	const want = ".pulsarules.yaml: the file must be UTF-8"
	testCases := []struct {
		name     string
		content  string
		isAccept bool
	}{
		{"utf-16 little endian with a byte order mark", "\xff\xfe%\x00Y\x00\n\x00", false},
		{"utf-16 big endian with a byte order mark", "\xfe\xff\x00%\x00Y\x00\n", false},
		{"utf-16 little endian directive", "\xff\xfev\x00e\x00r\x00", false},
		{"a nul byte", "version: 1\n\x00", false},
		{"a nul byte first", "\x00version: 1\n", false},
		{"an invalid byte", "version: 1\n\xff\n", false},
		{"a directive before an invalid byte", "%YAML\n\xff", false},
		{"a truncated character", "version: 1\n\xe2\x80", false},
		{"a utf-8 byte order mark", "\xef\xbb\xbfversion: 1\n", true},
		{"a multi-byte character", "version: 1\npreset: \"\u00e9\"\n", true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			_, err := decodeFile([]byte(testCase.content))
			if testCase.isAccept {
				if err != nil {
					t.Fatalf("decodeFile() error = %v, want nil", err)
				}
				return
			}
			if err == nil || err.Error() != want {
				t.Fatalf("decodeFile() error = %v, want %q", err, want)
			}
		})
	}
}

func TestDecodeAcceptsRealNullsAndMerges(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		content string
		preset  string
	}{
		{"tagged null spelled out", "version: 1\npreset: !!null null\n", ""},
		{"tagged null left empty", "version: 1\npreset: !!null\n", ""},
		{"bang merge key", "version: 1\n! <<: {preset: strict}\n", "strict"},
		{"tagged merge key", "version: 1\n!!merge <<: {preset: strict}\n", "strict"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			file, err := decodeFile([]byte(testCase.content))
			if err != nil || file.Preset != testCase.preset {
				t.Fatalf("decodeFile() = %+v, %v, want preset %q", file, err, testCase.preset)
			}
		})
	}
}
