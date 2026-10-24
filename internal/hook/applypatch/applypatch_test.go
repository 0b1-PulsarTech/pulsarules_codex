package applypatch

import (
	"slices"
	"testing"
)

type editCase struct {
	name  string
	patch string
	cwd   string
	want  []Edit
}

func runEditCases(t *testing.T, testCases []editCase) {
	t.Helper()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := Edits(testCase.patch, testCase.cwd)
			if !slices.Equal(got, testCase.want) {
				t.Errorf("Edits() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestEdits_Moves(t *testing.T) {
	t.Parallel()

	runEditCases(t, []editCase{
		{
			"move marks the source and the destination",
			"*** Update File: /abs/d.go\n*** Move to: /abs/e.go\n",
			"/work",
			[]Edit{{"/abs/d.go", MoveFrom}, {"/abs/e.go", MoveTo}},
		},
		{
			"move to after a hunk is not a move source",
			"*** Update File: /abs/d.go\n@@\n+x\n*** Move to: /abs/e.go\n",
			"/work",
			[]Edit{{"/abs/d.go", Update}, {"/abs/e.go", MoveTo}},
		},
	})
}

func TestEdits_Headers(t *testing.T) {
	t.Parallel()

	runEditCases(t, []editCase{
		{
			"indented file headers are trimmed",
			"  *** Add File: /a.go\n\t*** Delete File: /b.go\n   *** Update File: /c.go\n",
			"/work",
			[]Edit{{"/a.go", Add}, {"/b.go", Delete}, {"/c.go", Update}},
		},
		{
			"indented move to is not matched",
			"*** Update File: /a.go\n  *** Move to: /b.go\n",
			"/work",
			[]Edit{{"/a.go", Update}},
		},
	})
}

func TestEdits(t *testing.T) {
	t.Parallel()

	runEditCases(t, []editCase{
		{"empty patch", "", "/work", nil},
		{"junk", "hello\nworld\n*** Begin Patch\n", "/work", nil},
		{
			"add",
			"*** Begin Patch\n*** Add File: /abs/a.go\n+x\n*** End Patch",
			"/work",
			[]Edit{{"/abs/a.go", Add}},
		},
		{
			"update",
			"*** Update File: /abs/b.go\n@@\n-a\n+b\n",
			"/work",
			[]Edit{{"/abs/b.go", Update}},
		},
		{"delete", "*** Delete File: /abs/c.go\n", "/work", []Edit{{"/abs/c.go", Delete}}},
		{
			"multi file with duplicates",
			"*** Update File: /a.go\n*** Update File: /b.go\n*** Update File: /a.go\n",
			"/work",
			[]Edit{{"/a.go", Update}, {"/b.go", Update}},
		},
		{
			"crlf",
			"*** Add File: /a.go\r\n+x\r\n*** Delete File: /b.go\r\n",
			"/w",
			[]Edit{{"/a.go", Add}, {"/b.go", Delete}},
		},
		{
			"relative joins cwd",
			"*** Update File: pkg/a.go\n",
			"/work",
			[]Edit{{"/work/pkg/a.go", Update}},
		},
		{
			"relative without cwd stays",
			"*** Update File: pkg/a.go\n",
			"",
			[]Edit{{"pkg/a.go", Update}},
		},
		{
			"relative and absolute dedupe",
			"*** Update File: a.go\n*** Update File: /work/a.go\n",
			"/work",
			[]Edit{{"/work/a.go", Update}},
		},
		{"empty header path skipped", "*** Add File:   \n", "/work", nil},
		{"body line that looks like a header is not one", "+*** Add File: /x\n", "/work", nil},
	})
}

func TestOp_Gates(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		op       Op
		wantPre  bool
		wantPost bool
	}{
		{"", true, true},
		{Add, true, true},
		{Update, true, true},
		{Delete, false, false},
		{MoveFrom, true, false},
		{MoveTo, true, true},
	}
	for _, testCase := range testCases {
		t.Run(string(testCase.op), func(t *testing.T) {
			t.Parallel()

			if got := testCase.op.PreEdit(); got != testCase.wantPre {
				t.Errorf("PreEdit() = %v, want %v", got, testCase.wantPre)
			}
			if got := testCase.op.PostEdit(); got != testCase.wantPost {
				t.Errorf("PostEdit() = %v, want %v", got, testCase.wantPost)
			}
		})
	}
}
