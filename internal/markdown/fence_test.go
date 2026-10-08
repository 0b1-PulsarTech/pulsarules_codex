package markdown

import (
	"slices"
	"strings"
	"testing"
)

func TestCloses(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		info lineShape
		open fence
		want bool
	}{
		{"same column", lineShape{}, fence{}, true},
		{"three columns past a top-level opener", lineShape{indent: 3}, fence{}, true},
		{"four columns past a top-level opener", lineShape{indent: 4}, fence{}, false},
		{"a list fence closes at its own indent", lineShape{indent: 2}, fence{indent: 2}, true},
		{"a list fence closes up to column three", lineShape{indent: 3}, fence{indent: 2}, true},
		{"a list fence stays open past col three", lineShape{indent: 5}, fence{indent: 2}, false},
		{"a deep opener closes at its own indent", lineShape{indent: 6}, fence{indent: 6}, true},
		{"shallower than the opener", lineShape{indent: 1}, fence{indent: 2}, false},
		{"another quote depth", lineShape{depth: 1}, fence{}, false},
		{"the same quote depth", lineShape{depth: 1}, fence{depth: 1}, true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := closes(testCase.info, testCase.open); got != testCase.want {
				t.Errorf("closes(%+v, %+v) = %v", testCase.info, testCase.open, got)
			}
		})
	}
}

func TestReadingScan_RestartsAtAShallowerBareFence(t *testing.T) {
	t.Parallel()

	const src = "> ```go\n> x\n```\ny\n```\nz\n"
	scan := &readingScan{how: readings()[2]}
	offset := 0
	for _, text := range strings.SplitAfter(src, "\n") {
		scan.line(text, offset)
		offset += len(text)
	}
	want := []Region{{Start: 0, End: strings.Index(src, "z")}}
	if got := scan.finish(len(src)); !slices.Equal(got, want) {
		t.Errorf("regions = %v, want %v", got, want)
	}
}
