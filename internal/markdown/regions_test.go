package markdown

import (
	"slices"
	"testing"
)

func TestInRegions(t *testing.T) {
	t.Parallel()

	regions := []Region{{Start: 2, End: 5}, {Start: 7, End: 8}, {Start: 20, End: 30}}
	testCases := []struct {
		name   string
		offset int
		want   bool
	}{
		{"before the first region", 1, false},
		{"first byte of a region", 2, true},
		{"last byte of a region", 4, true},
		{"one past a region", 5, false},
		{"single byte region", 7, true},
		{"gap between regions", 10, false},
		{"inside the last region", 29, true},
		{"after the last region", 30, false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := inRegions(regions, testCase.offset); got != testCase.want {
				t.Errorf("inRegions(%d) = %v, want %v", testCase.offset, got, testCase.want)
			}
		})
	}
	if inRegions(nil, 0) {
		t.Error("no regions must read as prose")
	}
}

func TestUnionAll(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		lists [][]Region
		want  []Region
	}{
		{"no lists", nil, nil},
		{
			"disjoint lists interleave",
			[][]Region{{{0, 2}, {10, 12}}, {{4, 6}}},
			[]Region{{0, 2}, {4, 6}, {10, 12}},
		},
		{
			"overlapping and touching regions merge",
			[][]Region{{{0, 4}, {8, 9}}, {{3, 6}, {6, 8}}},
			[]Region{{0, 9}},
		},
		{"a contained region vanishes", [][]Region{{{0, 10}}, {{2, 3}}}, []Region{{0, 10}}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := unionAll(testCase.lists); !slices.Equal(got, testCase.want) {
				t.Errorf("unionAll = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestIntersectAll(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		lists [][]Region
		want  []Region
	}{
		{"no lists", nil, nil},
		{"one list is its own intersection", [][]Region{{{1, 3}}}, []Region{{1, 3}}},
		{"overlap keeps the shared part", [][]Region{{{0, 5}}, {{3, 9}}}, []Region{{3, 5}}},
		{"touching regions share nothing", [][]Region{{{0, 3}}, {{3, 6}}}, nil},
		{
			"a long region meets several short ones",
			[][]Region{{{0, 20}}, {{1, 2}, {5, 7}, {18, 25}}},
			[]Region{{1, 2}, {5, 7}, {18, 20}},
		},
		{"one empty list empties it", [][]Region{{{0, 5}}, nil}, nil},
		{
			"every list must agree",
			[][]Region{{{0, 10}}, {{2, 8}}, {{5, 20}}},
			[]Region{{5, 8}},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := intersectAll(testCase.lists); !slices.Equal(got, testCase.want) {
				t.Errorf("intersectAll = %v, want %v", got, testCase.want)
			}
		})
	}
}
