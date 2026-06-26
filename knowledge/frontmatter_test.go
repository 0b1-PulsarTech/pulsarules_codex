package knowledge

import "testing"

// TestSplitFrontmatter covers the fence parser: well-formed, body-empty, and the
// two missing-fence failure modes.
func TestSplitFrontmatter(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		wantErr  bool
		wantBody string
	}{
		{
			name:     "well-formed",
			input:    "---\nid: x\nname: X\n---\n# X\nbody line\n",
			wantBody: "# X\nbody line\n",
		},
		{
			name:     "no body",
			input:    "---\nid: x\nname: X\n---\n",
			wantBody: "",
		},
		{
			name:    "missing opening fence",
			input:   "# X\nbody\n",
			wantErr: true,
		},
		{
			name:    "missing closing fence",
			input:   "---\nid: x\nname: X\nbody without close\n",
			wantErr: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			meta, body, err := splitFrontmatter([]byte(testCase.input))
			if testCase.wantErr {
				if err == nil {
					t.Fatalf("expected error, got meta=%q body=%q", meta, body)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if body != testCase.wantBody {
				t.Errorf("body = %q, want %q", body, testCase.wantBody)
			}
		})
	}
}
