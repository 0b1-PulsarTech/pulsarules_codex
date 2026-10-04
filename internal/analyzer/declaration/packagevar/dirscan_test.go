package packagevar

import (
	"slices"
	"testing"
)

func TestAnalyzeDirScan(t *testing.T) {
	t.Parallel()

	runCases(t, slices.Concat(enumScopeCases(), []levelVarCase{
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
			name:  "Err constructor declared in an unchanged sibling exempt",
			files: map[string]string{"pkg/a.go": "package p\n\nvar ErrX = newErr(\"x\")\n"},
			extra: map[string]string{
				"pkg/b.go": "package p\n\nfunc newErr(s string) error { return nil }\n",
			},
		},
		{
			name:  "Err constructor returning a non error in a sibling flagged",
			files: map[string]string{"pkg/a.go": "package p\n\nvar ErrX = newErr(\"x\")\n"},
			extra: map[string]string{
				"pkg/b.go": "package p\n\nfunc newErr(s string) int { return 0 }\n",
			},
			want: []string{"pkg/a.go:3:package-level var ErrX"},
		},
		{
			name:  "Err constructor method in a sibling does not count",
			files: map[string]string{"pkg/a.go": "package p\n\nvar ErrX = newErr(\"x\")\n"},
			extra: map[string]string{
				"pkg/b.go": "package p\n\ntype s struct{}\n\n" +
					"func (s) newErr(string) error { return nil }\n",
			},
			want: []string{"pkg/a.go:3:package-level var ErrX"},
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
	}))
}

func enumScopeCases() []levelVarCase {
	const enumScopeSrc = "package p\n\nimport \"x/permission\"\n\nvar (\n" +
		"\tPermView = permission.New(keyView)\n\tPermManage = permission.New(keyManage)\n)\n"

	return []levelVarCase{
		{
			name: "consts of the analysed file exempt",
			files: map[string]string{
				"pkg/a.go": enumScopeSrc + "\nconst keyView, keyManage = \"v\", \"m\"\n",
			},
		},
		{
			name: "const inside a function body does not count",
			files: map[string]string{
				"pkg/a.go": enumScopeSrc +
					"\nfunc f() {\n\tconst keyView, keyManage = \"v\", \"m\"\n}\n",
			},
			want: []string{
				"pkg/a.go:6:package-level var PermView",
				"pkg/a.go:7:package-level var PermManage",
			},
		},
		{
			name:  "consts only in an external test package do not count",
			files: map[string]string{"pkg/a.go": enumScopeSrc},
			extra: map[string]string{
				"pkg/a_x_test.go": "package p_test\n\nconst keyView, keyManage = \"v\", \"m\"\n",
			},
			want: []string{
				"pkg/a.go:6:package-level var PermView",
				"pkg/a.go:7:package-level var PermManage",
			},
		},
		{
			name:  "consts in another package of the same dir do not count",
			files: map[string]string{"pkg/a.go": enumScopeSrc},
			extra: map[string]string{
				"pkg/b.go": "package q\n\nconst keyView, keyManage = \"v\", \"m\"\n",
			},
			want: []string{
				"pkg/a.go:6:package-level var PermView",
				"pkg/a.go:7:package-level var PermManage",
			},
		},
	}
}
