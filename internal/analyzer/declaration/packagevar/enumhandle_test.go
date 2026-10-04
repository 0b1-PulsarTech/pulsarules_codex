package packagevar

import (
	"slices"
	"testing"
)

const enumSchema = "var Schema = permission.DefineModule[moduleMod](keyView)\n"

func enumHandleExemptCases() []levelVarCase {
	return []levelVarCase{
		{
			name: "single handle of a constant schema exempt",
			files: map[string]string{"a.go": enumSrc(
				enumConsts+enumSchema,
				"var PermView = Schema.Perm(keyView)",
			)},
		},
		{
			name: "unparenthesised handles of a constant schema exempt",
			files: map[string]string{"a.go": enumSrc(
				enumConsts+enumSchema,
				"var PermView = Schema.Perm(keyView)\n",
				"var PermManage = Schema.Perm(keyManage)",
			)},
		},
		{
			name: "generic handle of a constant schema exempt",
			files: map[string]string{"a.go": enumSrc(
				enumConsts+enumSchema,
				"var PermView = Schema.Perm[moduleMod](keyView)",
			)},
		},
	}
}

func enumHandleReportedCases() []levelVarCase {
	return []levelVarCase{
		{
			name: "handle of a handle is reported",
			files: map[string]string{"a.go": enumSrc(
				enumConsts+enumSchema,
				"var Sub = Schema.Perm(keyView)\n",
				"var PermView = Sub.Perm(keyManage)",
			)},
			want: []string{"a.go:19:package-level var PermView"},
		},
		{
			name: "handle with a non constant argument is reported",
			files: map[string]string{"a.go": enumSrc(
				enumConsts+"var dynamic = 1\n",
				enumSchema,
				"var PermView = Schema.Perm(dynamic)",
			)},
			want: []string{
				"a.go:15:package-level var dynamic",
				"a.go:17:package-level var Schema",
				"a.go:19:package-level var PermView",
			},
		},
		{
			name: "handle without arguments is reported",
			files: map[string]string{"a.go": enumSrc(
				enumConsts+enumSchema,
				"var PermView = Schema.Perm()",
			)},
			want: []string{
				"a.go:15:package-level var Schema",
				"a.go:17:package-level var PermView",
			},
		},
		{
			name: "handle of a schema with a non constant argument is reported",
			files: map[string]string{"a.go": enumSrc(
				enumConsts+"var dynamic = 1\n",
				"var Schema = permission.DefineModule[moduleMod](dynamic)\n",
				"var PermView = Schema.Perm(keyView)",
			)},
			want: []string{
				"a.go:15:package-level var dynamic",
				"a.go:17:package-level var Schema",
				"a.go:19:package-level var PermView",
			},
		},
		{
			name: "handle of a schema from a local function is reported",
			files: map[string]string{"a.go": enumSrc(
				enumConsts+"var Schema = define(keyView)\n",
				"var PermView = Schema.Perm(keyView)",
			)},
			want: []string{
				"a.go:15:package-level var Schema",
				"a.go:17:package-level var PermView",
			},
		},
		{
			name: "lone package call is reported",
			files: map[string]string{"a.go": enumSrc(
				enumConsts + "var PermView = permission.New(keyView)",
			)},
			want: []string{"a.go:15:package-level var PermView"},
		},
	}
}

func enumHandleScopeCases() []levelVarCase {
	return []levelVarCase{
		{
			name: "multi name handle is reported",
			files: map[string]string{"a.go": enumSrc(
				enumConsts+enumSchema,
				"var PermView, extra = Schema.Perm(keyView)",
			)},
			want: []string{
				"a.go:15:package-level var Schema",
				"a.go:17:package-level var PermView",
				"a.go:17:package-level var extra",
			},
		},
		{
			name: "multi name schema is reported with its handle",
			files: map[string]string{"a.go": enumSrc(
				enumConsts+"var Schema, extra = permission.Define(keyView)\n",
				"var PermView = Schema.Perm(keyView)",
			)},
			want: []string{
				"a.go:15:package-level var Schema",
				"a.go:15:package-level var extra",
				"a.go:17:package-level var PermView",
			},
		},
		{
			name: "handle of a schema in another file is reported",
			files: map[string]string{"a.go": enumSrc(
				enumConsts + "var PermView = Schema.Perm(keyView)",
			)},
			extra: map[string]string{"b.go": "package p\n\n" + enumSchema},
			want:  []string{"a.go:15:package-level var PermView"},
		},
		{
			name: "blank handle does not exempt the schema",
			files: map[string]string{"a.go": enumSrc(
				enumConsts+enumSchema,
				"var _ = Schema.Perm(keyView)",
			)},
			want: []string{
				"a.go:15:package-level var Schema",
				"a.go:17:package-level var _",
			},
		},
		{
			name: "block of non constant handles leaves the schema reported",
			files: map[string]string{"a.go": enumSrc(
				enumConsts+"var dynamic = 1\n",
				enumSchema,
				"var (",
				"\tPermView   = Schema.Perm(dynamic)",
				"\tPermManage = Schema.Perm(dynamic + 1)",
				")",
			)},
			want: []string{
				"a.go:15:package-level var dynamic",
				"a.go:17:package-level var Schema",
				"a.go:20:package-level var PermView",
				"a.go:21:package-level var PermManage",
			},
		},
	}
}

func TestAnalyzeEnumHandle(t *testing.T) {
	t.Parallel()

	runCases(t, slices.Concat(
		enumHandleExemptCases(),
		enumHandleReportedCases(),
		enumHandleScopeCases(),
	))
}
