package packagelevelvar

import (
	"slices"
	"strings"
	"testing"
)

const enumConsts = "const (\n\tkeyView   = \"view\"\n\tkeyManage = \"manage\"\n)\n\n"

func enumSrc(lines ...string) string {
	const header = "package p\n\nimport (\n\t\"errors\"\n\t\"regexp\"\n\n\t\"x/permitek\"\n)\n\n"

	return header + strings.Join(lines, "\n") + "\n"
}

func enumExemptCases() []levelVarCase {
	return append(enumExemptDeclCases(), enumExemptConstArgCases()...)
}

func enumExemptDeclCases() []levelVarCase {
	return []levelVarCase{
		{
			name: "schema and handle block exempt",
			files: map[string]string{"a.go": enumSrc(
				enumConsts+"var Schema = permitek.DefineModule[moduleMod](keyView, keyManage)\n",
				"var (",
				"\tPermView   = Schema.Perm(keyView)",
				"\tPermManage = Schema.Perm(keyManage)",
				")",
			)},
		},
		{
			name: "consts declared in a sibling file",
			files: map[string]string{"a.go": enumSrc(
				"var (",
				"\tPermView   = permitek.New(keyView)",
				"\tPermManage = permitek.New(keyManage)",
				")",
			)},
			extra: map[string]string{"c.go": "package p\n\n" + enumConsts},
		},
		{
			name: "generic with one type argument",
			files: map[string]string{"a.go": enumSrc(
				enumConsts+"var (",
				"\tPermView   = permitek.New[View](keyView)",
				"\tPermManage = permitek.New[View](keyManage)",
				")",
			)},
		},
		{
			name: "generic with two type arguments",
			files: map[string]string{"a.go": enumSrc(
				enumConsts+"var (",
				"\tPermView   = permitek.New[View, Kind](keyView)",
				"\tPermManage = permitek.New[View, Kind](keyManage)",
				")",
			)},
		},
	}
}

func enumExemptConstArgCases() []levelVarCase {
	return []levelVarCase{
		{
			name: "binary and paren constant arguments",
			files: map[string]string{"a.go": enumSrc(
				enumConsts+"var (",
				"\tPermView   = permitek.New((keyView + \"_x\"), -2)",
				"\tPermManage = permitek.New(keyManage+\"_y\", 3*(1+2))",
				")",
			)},
		},
		{
			name: "explicit spec type",
			files: map[string]string{"a.go": enumSrc(
				enumConsts+"var (",
				"\tPermView   Perm = permitek.New(keyView)",
				"\tPermManage Perm = permitek.New(keyManage)",
				")",
			)},
		},
		{
			name: "sentinel still exempt beside a group",
			files: map[string]string{"a.go": enumSrc(
				enumConsts+"var ErrX = errors.New(\"x\")\n",
				"var (",
				"\tPermView   = permitek.New(keyView)",
				"\tPermManage = permitek.New(keyManage)",
				")",
			)},
		},
		{
			name: "two constant regexp compiles exempt by design",
			files: map[string]string{"a.go": enumSrc(
				"var (",
				"\tfirst  = regexp.MustCompile(\"a\")",
				"\tsecond = regexp.MustCompile(\"b\")",
				")",
			)},
		},
	}
}

func enumReportedCases() []levelVarCase {
	return []levelVarCase{
		{
			name: "single spec block",
			files: map[string]string{"a.go": enumSrc(
				enumConsts+"var (",
				"\tPermView = permitek.New(keyView)",
				")",
			)},
			want: []string{"a.go:16:package-level var PermView"},
		},
		{
			name: "unparenthesised single var",
			files: map[string]string{"a.go": enumSrc(
				enumConsts + "var PermView = permitek.New(keyView)",
			)},
			want: []string{"a.go:15:package-level var PermView"},
		},
		{
			name: "mixed callees",
			files: map[string]string{"a.go": enumSrc(
				enumConsts+"var (",
				"\tPermView   = permitek.New(keyView)",
				"\tPermManage = permitek.Other(keyManage)",
				")",
			)},
			want: []string{
				"a.go:16:package-level var PermView",
				"a.go:17:package-level var PermManage",
			},
		},
		{
			name: "var argument",
			files: map[string]string{"a.go": enumSrc(
				enumConsts+"var dynamic = \"x\"\n",
				"var (",
				"\tPermView   = permitek.New(keyView)",
				"\tPermManage = permitek.New(dynamic)",
				")",
			)},
			want: []string{
				"a.go:15:package-level var dynamic",
				"a.go:18:package-level var PermView",
				"a.go:19:package-level var PermManage",
			},
		},
		{
			name: "selector argument",
			files: map[string]string{"a.go": enumSrc(
				enumConsts+"var (",
				"\tPermView   = permitek.New(keyView)",
				"\tPermManage = permitek.New(permitek.Manage)",
				")",
			)},
			want: []string{
				"a.go:16:package-level var PermView",
				"a.go:17:package-level var PermManage",
			},
		},
	}
}

func enumArgCases() []levelVarCase {
	return []levelVarCase{
		{
			name: "call argument",
			files: map[string]string{"a.go": enumSrc(
				enumConsts+"var (",
				"\tPermView   = permitek.New(keyView)",
				"\tPermManage = permitek.New(name())",
				")",
			)},
			want: []string{
				"a.go:16:package-level var PermView",
				"a.go:17:package-level var PermManage",
			},
		},
		{
			name: "zero arguments",
			files: map[string]string{"a.go": enumSrc(
				enumConsts+"var (",
				"\tPermView   = permitek.New(keyView)",
				"\tPermManage = permitek.New()",
				")",
			)},
			want: []string{
				"a.go:16:package-level var PermView",
				"a.go:17:package-level var PermManage",
			},
		},
	}
}

func enumShapeCases() []levelVarCase {
	return []levelVarCase{
		{
			name: "blank name",
			files: map[string]string{"a.go": enumSrc(
				enumConsts+"var (",
				"\tPermView = permitek.New(keyView)",
				"\t_        = permitek.New(keyManage)",
				")",
			)},
			want: []string{
				"a.go:16:package-level var PermView",
				"a.go:17:package-level var _",
			},
		},
		{
			name: "multi name spec",
			files: map[string]string{"a.go": enumSrc(
				enumConsts+"var (",
				"\tPermView           = permitek.New(keyView)",
				"\tfirst, second      = permitek.New(keyManage), permitek.New(keyView)",
				")",
			)},
			want: []string{
				"a.go:16:package-level var PermView",
				"a.go:17:package-level var first",
				"a.go:17:package-level var second",
			},
		},
		{
			name: "receiver is not a package var",
			files: map[string]string{"a.go": enumSrc(
				enumConsts+"var (",
				"\tPermView   = local.Perm(keyView)",
				"\tPermManage = local.Perm(keyManage)",
				")",
			)},
			want: []string{
				"a.go:16:package-level var PermView",
				"a.go:17:package-level var PermManage",
			},
		},
		{
			name: "receiver declared in another file",
			files: map[string]string{"a.go": enumSrc(
				enumConsts+"var (",
				"\tPermView   = Schema.Perm(keyView)",
				"\tPermManage = Schema.Perm(keyManage)",
				")",
			)},
			extra: map[string]string{
				"b.go": "package p\n\nvar Schema = permitek.Define(keyView)\n",
			},
			want: []string{
				"a.go:16:package-level var PermView",
				"a.go:17:package-level var PermManage",
			},
		},
	}
}

func enumDefinitionCases() []levelVarCase {
	return []levelVarCase{
		{
			name: "schema with non constant argument is reported, handles exempt",
			files: map[string]string{"a.go": enumSrc(
				enumConsts+"var dynamic = \"x\"\n",
				"var Schema = permitek.Define(dynamic)\n",
				"var (",
				"\tPermView   = Schema.Perm(keyView)",
				"\tPermManage = Schema.Perm(keyManage)",
				")",
			)},
			want: []string{
				"a.go:15:package-level var dynamic",
				"a.go:17:package-level var Schema",
			},
		},
		{
			name: "schema without arguments is reported, handles exempt",
			files: map[string]string{"a.go": enumSrc(
				enumConsts+"var Schema = permitek.Define()\n",
				"var (",
				"\tPermView   = Schema.Perm(keyView)",
				"\tPermManage = Schema.Perm(keyManage)",
				")",
			)},
			want: []string{"a.go:15:package-level var Schema"},
		},
		{
			name: "schema from a local function is reported",
			files: map[string]string{"a.go": enumSrc(
				enumConsts+"var Schema = define(keyView)\n",
				"var (",
				"\tPermView   = Schema.Perm(keyView)",
				"\tPermManage = Schema.Perm(keyManage)",
				")",
			)},
			want: []string{"a.go:15:package-level var Schema"},
		},
		{
			name: "constant schema unused as a receiver is reported",
			files: map[string]string{"a.go": enumSrc(
				enumConsts + "var Schema = permitek.Define(keyView)",
			)},
			want: []string{"a.go:15:package-level var Schema"},
		},
		{
			name: "lone regexp compile is reported",
			files: map[string]string{"a.go": enumSrc(
				"var pattern = regexp.MustCompile(\"x\")",
			)},
			want: []string{"a.go:10:package-level var pattern"},
		},
	}
}

func enumReceiverCases() []levelVarCase {
	return []levelVarCase{
		{
			name: "receiver declared with a type only is reported, handles exempt",
			files: map[string]string{"a.go": enumSrc(
				enumConsts+"var Schema permitek.Module\n",
				"var (",
				"\tPermView   = Schema.Perm(keyView)",
				"\tPermManage = Schema.Perm(keyManage)",
				")",
			)},
			want: []string{"a.go:15:package-level var Schema"},
		},
		{
			name: "receiver declared in a multi name spec is reported, handles exempt",
			files: map[string]string{"a.go": enumSrc(
				enumConsts+"var Schema, Other = permitek.A(keyView), permitek.B(keyView)\n",
				"var (",
				"\tPermView   = Schema.Perm(keyView)",
				"\tPermManage = Schema.Perm(keyManage)",
				")",
			)},
			want: []string{
				"a.go:15:package-level var Schema",
				"a.go:15:package-level var Other",
			},
		},
		{
			name: "receiver declared inside a group is exempt with its handles",
			files: map[string]string{"a.go": enumSrc(
				enumConsts+"var (\n\tSchema = permitek.Define(keyView)\n\tOther = 1\n)\n",
				"var (",
				"\tPermView   = Schema.Perm(keyView)",
				"\tPermManage = Schema.Perm(keyManage)",
				")",
			)},
			want: []string{"a.go:17:package-level var Other"},
		},
		{
			name: "receiver built from another package var is reported",
			files: map[string]string{"a.go": enumSrc(
				enumConsts+"var Base permitek.Module\nvar Schema = Base.Sub(keyView)\n",
				"var (",
				"\tPermView   = Schema.Perm(keyView)",
				"\tPermManage = Schema.Perm(keyManage)",
				")",
			)},
			want: []string{
				"a.go:15:package-level var Base",
				"a.go:16:package-level var Schema",
			},
		},
	}
}

func enumOperandCases() []levelVarCase {
	return []levelVarCase{
		{
			name: "two names with one multi return value is not a group",
			files: map[string]string{"a.go": enumSrc(
				enumConsts+"var (",
				"\tPermView = permitek.New(keyView)",
				"\tfirst, second = permitek.New(keyManage)",
				")",
			)},
			want: []string{
				"a.go:16:package-level var PermView",
				"a.go:17:package-level var first",
				"a.go:17:package-level var second",
			},
		},
		{
			name: "binary argument with a non constant right operand",
			files: map[string]string{"a.go": enumSrc(
				enumConsts+"var dynamic = \"x\"\n",
				"var (",
				"\tPermView   = permitek.New(keyView)",
				"\tPermManage = permitek.New(keyManage + dynamic)",
				")",
			)},
			want: []string{
				"a.go:15:package-level var dynamic",
				"a.go:18:package-level var PermView",
				"a.go:19:package-level var PermManage",
			},
		},
		dynamicArgCase("binary argument with a non constant left operand", "dynamic + keyManage"),
		dynamicArgCase("paren around a non constant argument", "(dynamic)"),
		dynamicArgCase("unary on a non constant argument", "-dynamic"),
		{
			name: "constant schema beside an invalid group is reported",
			files: map[string]string{"a.go": enumSrc(
				enumConsts+"var dynamic = 1\n",
				"var Schema = permitek.Define(keyView)\n",
				"var (",
				"\tPermView   = Schema.Perm(keyView)",
				"\tPermManage = Schema.Perm(dynamic)",
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

func dynamicArgCase(name, arg string) levelVarCase {
	return levelVarCase{
		name: name,
		files: map[string]string{"a.go": enumSrc(
			enumConsts+"var dynamic = 1\n",
			"var (",
			"\tPermView   = permitek.New(keyView)",
			"\tPermManage = permitek.New("+arg+")",
			")",
		)},
		want: []string{
			"a.go:15:package-level var dynamic",
			"a.go:18:package-level var PermView",
			"a.go:19:package-level var PermManage",
		},
	}
}

func TestAnalyzeEnumGroup(t *testing.T) {
	t.Parallel()

	runCases(t, slices.Concat(
		enumExemptCases(),
		enumReportedCases(),
		enumArgCases(),
		enumShapeCases(),
		enumDefinitionCases(),
		enumReceiverCases(),
		enumOperandCases(),
	))
}
