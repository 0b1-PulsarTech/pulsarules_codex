package sfc

import (
	"slices"
	"testing"
)

func TestRegions(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		text string
		want []Region
	}{
		{"no element", "<p>a</p>", nil},
		{"script body", "<script>x</script>", []Region{{Tag: "script", Start: 8, End: 9}}},
		{"style body", "<style>a</style>", []Region{{Tag: "style", Start: 7, End: 8}}},
		{
			"script and style in order",
			"<script>a</script><style>b</style>",
			[]Region{{Tag: "script", Start: 8, End: 9}, {Tag: "style", Start: 25, End: 26}},
		},
		{"script named in a comment", "<!-- <script>x</script> -->", nil},
		{"markup comment before a script", "<!-- a --><script>x</script>", []Region{
			{Tag: "script", Start: 18, End: 19},
		}},
		{"unterminated comment hides the rest", "<!-- a <script>x</script>", nil},
		{"unterminated script runs to the end", "<script>x", []Region{
			{Tag: "script", Start: 8, End: 9},
		}},
		{"attributes on their own line", "<script\n lang=\"ts\"\n>x</script>", []Region{
			{Tag: "script", Start: 20, End: 21},
		}},
		{"quoted bracket inside an attribute", "<script a=\">\">x</script>", []Region{
			{Tag: "script", Start: 14, End: 15},
		}},
		{"end tag needs a boundary", "<script>a</scripts>b</script>", []Region{
			{Tag: "script", Start: 8, End: 20},
		}},
		{
			"end tag with a blank",
			"<script>a</script >",
			[]Region{{Tag: "script", Start: 8, End: 9}},
		},
		{"script inside an element", "<div><script>x</script></div>", []Region{
			{Tag: "script", Start: 13, End: 14},
		}},
		{"upper-case tag is a component", "<SCRIPT>x</SCRIPT>", nil},
		{"self-closing script has no body", "<script/>x", nil},
		{"stray brace stops the markup scan", "{<script>x</script>", nil},
		{"balanced mustache is skipped", "{a}<script>x</script>", []Region{
			{Tag: "script", Start: 11, End: 12},
		}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := Regions(testCase.text); !slices.Equal(got, testCase.want) {
				t.Fatalf("Regions(%q) = %v, want %v", testCase.text, got, testCase.want)
			}
		})
	}
}

func TestPlainRegions(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		text string
		want []Region
	}{
		{"stray brace does not stop the scan", "<p>{</p><script>x</script>", []Region{
			{Tag: "script", Start: 16, End: 17},
		}},
		{"brace in an attribute is plain", "<p a={><script>x</script>", []Region{
			{Tag: "script", Start: 15, End: 16},
		}},
		{"script named in a comment", "<!-- <script>x</script> -->", nil},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := PlainRegions(testCase.text); !slices.Equal(got, testCase.want) {
				t.Fatalf("PlainRegions(%q) = %v, want %v", testCase.text, got, testCase.want)
			}
		})
	}
}
