package packagelevelvar

import "testing"

func TestAnalyzeDirScan(t *testing.T) {
	t.Parallel()

	runCases(t, []levelVarCase{
		{
			name: "conversion to a type declared in an unchanged sibling exempt",
			files: map[string]string{
				"pkg/a.go": "package p\n\nvar _ I = handler(nil)\n\ntype I interface{}\n",
			},
			extra: map[string]string{"pkg/b.go": "package p\n\ntype handler func()\n"},
		},
		{
			name:  "type in an unchanged file of another dir does not exempt",
			files: map[string]string{"pkg/a.go": "package p\n\nvar _ = handler(nil)\n"},
			extra: map[string]string{"other/b.go": "package p\n\ntype handler func()\n"},
			want:  []string{"pkg/a.go:3:package-level var _"},
		},
		{
			name: "type in an external test package does not exempt",
			files: map[string]string{
				"pkg/a.go": "package p\n\nfunc load(x int) int { return x }\n\n" +
					"type I interface{}\n\ntype T struct{ v int }\n\nvar _ I = T{v: load(1)}\n",
			},
			extra: map[string]string{"pkg/a_x_test.go": "package p_test\n\ntype load int\n"},
			want:  []string{"pkg/a.go:9:package-level var _"},
		},
		{
			name:  "unparsable sibling is skipped",
			files: map[string]string{"pkg/a.go": "package p\n\nvar _ = handler(nil)\n"},
			extra: map[string]string{"pkg/b.go": "package p\n\ntype handler func(\n"},
			want:  []string{"pkg/a.go:3:package-level var _"},
		},
	})
}
