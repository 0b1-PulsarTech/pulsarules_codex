package sandbox

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestFrameRoundTrip(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	want := Request{
		Seq:    3,
		Source: syntax.Source{Path: "a.ts", Content: []byte("let a;"), IsTest: true},
		Local:  []string{"#lib/*"},
	}
	if err := writeFrame(&buf, want); err != nil {
		t.Fatalf("write: %v", err)
	}
	var got Request
	if err := readFrame(&buf, &got); err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("read %+v, %v", got, err)
	}
	if err := readFrame(&buf, &got); !errors.Is(err, io.EOF) {
		t.Fatalf("drained read: %v", err)
	}
}

func TestFrameKeepsBytes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		path string
		text string
	}{
		{name: "non-utf8 path", path: "bad\xffname.ts", text: "let a;"},
		{name: "latin-1 body", path: "a.ts", text: "const caf\xe9 = \"\xff\xfe\";"},
		{name: "nul and bom", path: "a.ts", text: "\x00\xef\xbb\xbf\xc0\x80"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			want := Request{
				Seq:    1,
				Source: syntax.Source{Path: testCase.path, Content: []byte(testCase.text)},
			}
			if err := writeFrame(&buf, want); err != nil {
				t.Fatalf("write: %v", err)
			}
			var got Request
			if err := readFrame(&buf, &got); err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("read %+v, %v", got, err)
			}
		})
	}
}

func TestReadFrameFailures(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		input   []byte
		wantErr error
		notEOF  bool
	}{
		{name: "empty", input: nil, wantErr: io.EOF},
		{name: "short header", input: []byte{0, 0}, wantErr: io.ErrUnexpectedEOF},
		{name: "short body", input: []byte{0, 0, 0, 9, '{'}, wantErr: io.ErrUnexpectedEOF},
		{name: "no body", input: []byte{0, 0, 0, 9}, wantErr: io.ErrUnexpectedEOF, notEOF: true},
		{name: "oversize", input: []byte{0xff, 0xff, 0xff, 0xff}, wantErr: errFrameSize},
		{name: "bad gob", input: []byte{0, 0, 0, 2, '{', '{'}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var got Request
			err := readFrame(bytes.NewReader(testCase.input), &got)
			if err == nil {
				t.Fatal("no error")
			}
			if testCase.wantErr != nil && !errors.Is(err, testCase.wantErr) {
				t.Fatalf("error %v, want %v", err, testCase.wantErr)
			}
			if testCase.notEOF && errors.Is(err, io.EOF) {
				t.Fatalf("truncated frame read as clean end: %v", err)
			}
		})
	}
}

func TestWriteFrameFailures(t *testing.T) {
	t.Parallel()

	if err := writeFrame(failWriter{}, Request{}); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("closed writer: %v", err)
	}
	if err := writeFrame(io.Discard, make(chan int)); err == nil {
		t.Fatal("unencodable payload accepted")
	}
	huge := Request{Source: syntax.Source{Content: []byte(strings.Repeat("a", maxFrame))}}
	if err := writeFrame(io.Discard, huge); !errors.Is(err, errFrameSize) {
		t.Fatalf("oversize payload: %v", err)
	}
}
