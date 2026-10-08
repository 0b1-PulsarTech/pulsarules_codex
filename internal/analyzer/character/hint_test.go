package character

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
)

func rawHint(escape string) string {
	return "a raw string cannot hold an escape: move it into an interpreted string as " + escape
}

func raw(name, rel, body string) hintCase {
	return outside(name, rel, body, rawHint("\\u00A0"))
}

func escaped(name, rel, body string) hintCase {
	return inLiteral(name, rel, body, "\\u00A0")
}

func rawCases() []hintCase {
	return []hintCase{
		raw("go raw string", "a.go", "var a = `x\u00A0y`\n"),
		raw("go raw with quotes", "a.go", "var a = `say \"x\u00A0\" // y /* z`\n"),
		raw("go raw with rune quote", "a.go", "var a = `say 'x\u00A0' // y`\n"),
		raw("go raw multiline", "a.go", "var a = `x\n\u00A0y`\n"),
		raw("dart raw single", "a.dart", "var a = r'x\u00A0y';\n"),
		raw("dart raw double", "a.dart", "var a = r\"x\u00A0y\";\n"),
		raw("dart raw triple", "a.dart", "var a = r\"\"\"x\n\u00A0\"\"\";\n"),
		raw("dart raw triple single", "a.dart", "var a = r'''x\u00A0''';\n"),
		raw("mjs String.raw", "a.mjs", "const a = String.raw`x\u00A0y`;\n"),
		raw("mjs String.raw then a blank", "a.mjs", "const a = String.raw `x\u00A0y`;\n"),
		raw("mjs String.raw then a newline", "a.mjs", "const a = String.raw\n`x\u00A0y`;\n"),
		escaped("dart plain is not raw", "a.dart", "var a = 'x\u00A0y';\n"),
		escaped("mjs template is not raw", "a.mjs", "const a = `x\u00A0y`;\n"),
	}
}

func regexCases() []hintCase {
	const (
		byHand = "remove it by hand"
		astral = "\\u{E0001}"
		money  = "const money = `R$\u00A010`;\n"
	)
	return []hintCase{
		escaped("mjs regex body", "a.mjs", "const re = /x\u00A0y/;\n"),
		escaped("mjs regex with quotes", "a.mjs", "const re = /value=\"([^\"]*)\u00A0\"/;\n"),
		escaped("mjs regex after paren", "a.mjs", "m.matchAll(/<li>\u00A0/g)\n"),
		escaped("mjs string after a regex", "a.mjs", "s.replace(/\"/g, \"R$\u00A010\")\n"),
		escaped("mjs backtick in a regex", "a.mjs", "x = /`\u00A0/;\n"),
		outside("mjs division is code", "a.mjs", "x = a / b\u00A0/ c;\n", byHand),
		{
			name: "mjs astral regex needs the u flag",
			rel:  "a.mjs",
			body: "const re = /x\U000E0001/;\n",
			hint: "write it as the escape \\u{E0001} and add the u flag",
			sev:  finding.SeverityWarning,
		},
		inLiteralWarn("mjs astral string", "a.mjs", "const a = '\U000E0001';\n", astral),
		escaped("svelte regex", "a.svelte", "<script>x = /a\u00A0/;</script>\n"),
		escaped("mjs regex after return", "a.mjs", "f = () => { return /`/.test(s); }\n"+money),
		escaped("mjs regex after an if header", "a.mjs", "if (ok) /`/.test(s);\n"+money),
		escaped("mjs division after a call", "a.mjs", "h = f(x) / 2 + `a\n`;\n"+money),
		escaped("mjs regex after a bang", "a.mjs", "ok = !/`/.test(s);\n"+money),
		escaped("mjs postfix increment", "a.mjs", "let i = 0;\nh = i++ / 2 + `a\n`;\n"+money),
		escaped("mjs property keyword", "a.mjs", "h = node.return / 2 + `a\n`;\n"+money),
		escaped("mjs line start slash", "a.mjs", "h = total\n  / count + `a\nb`;\n"+money),
		escaped("ts non-null slash", "a.ts", "p = `${done! / total}%`;\nt = `a`;\n"+money),
	}
}

func TestHint_Raw(t *testing.T) {
	t.Parallel()
	runHintCases(t, rawCases())
}

func TestHint_Regex(t *testing.T) {
	t.Parallel()
	runHintCases(t, regexCases())
}

func TestHint_UnscannedLanguages(t *testing.T) {
	t.Parallel()

	const (
		carrier = "remove it by hand; inside a string literal it is data, " +
			"so write the language's escape instead"
		prose = "replace it with the ASCII form by hand; inside a string literal it is data, " +
			"so write the language's escape instead"
	)
	testCases := []hintCase{
		outside("svelte markup text", "a.svelte", "<p>it's\u00A0here</p>\n", carrier),
		outside("svelte markup expression", "a.svelte", "<p>{'R$\u00A0'}</p>\n", carrier),
		outside(
			"svelte attribute",
			"a.svelte",
			"<script>let a = 1;</script>\n<input placeholder=\"R$\u00A00,00\" />\n",
			carrier,
		),
		outside(
			"vue template", "a.vue", "<template><p>R$\u00A0</p></template>\n", carrier,
		),
		typographic("svelte markup typographic", "a.svelte", "<p>wait\u2026</p>\n", prose),
		outside("rust string", "a.rs", "let a = \"x\u00A0y\";\n", carrier),
		outside("php string", "a.php", "$a = 'x\u00A0y';\n", carrier),
		outside("java string", "a.java", "String a = \"x\u00A0y\";\n", carrier),
		outside("python string", "a.py", "a = 'x\u00A0y'\n", carrier),
		typographic("rust typographic", "a.rs", "let a = \"x\u2026\";\n", prose),
		typographic("java typographic", "a.java", "String a = \"x\u2026\";\n", prose),
	}
	runHintCases(t, testCases)
}

func TestHint_UnlexableFileIsUnscanned(t *testing.T) {
	t.Parallel()

	const (
		carrier = "remove it by hand; could not read this file's strings; " +
			"inside a string literal it is data, so write the language's escape instead"
		prose = "replace it with the ASCII form by hand; could not read this file's " +
			"strings; inside a string literal it is data, " +
			"so write the language's escape instead"
	)
	testCases := []hintCase{
		outside("go cut-off comment", "a.go", "/* x\nvar a = \"R$\u00A010\"\n", carrier),
		outside(
			"go odd backtick", "a.go", "var u = `h\nvar a = \"R$\u00A010\"\n", carrier,
		),
		outside("mjs cut-off template", "a.mjs", "const a = `R$\u00A010 ${b\n", carrier),
		typographic(
			"go cut-off comment ellipsis",
			"a.go",
			"/* x\nvar a = \"wait\u2026\"\n",
			prose,
		),
	}
	runHintCases(t, testCases)
}

func jsx(price, apostrophe, after string) string {
	return "const price = \"R$" + price + "10\";\nexport const A = () => (\n  <p>\n" +
		"    Don't" + apostrophe + "show\n  </p>\n" + after + ");\n"
}

func TestHint_LineCutKeepsTheStringsBeforeIt(t *testing.T) {
	t.Parallel()

	const past = "; could not read this file's strings from line 4 on; " +
		"inside a string literal it is data, so write the language's escape instead"
	const before = "; could not read this file's strings reliably: a literal on line 4 never " +
		"closes; inside a string literal it is data, so write the language's escape instead"
	testCases := []hintCase{
		inLiteral("string before the cut", "a.tsx", jsx("\u00A0", " ", ""), "\\u00A0"),
		outside(
			"marker on the cut line",
			"a.tsx",
			jsx(" ", "\u00A0", ""),
			"remove it by hand"+past,
		),
		outside(
			"marker after the cut line",
			"a.tsx",
			jsx(" ", " ", "\u00A0"),
			"remove it by hand"+past,
		),
		typographic(
			"typographic after the cut", "a.tsx", jsx(" ", "\u2026", ""),
			"replace it with the ASCII form by hand"+past,
		),
		outside(
			"marker in a comment before the cut",
			"a.tsx",
			"// x\u00A0y\nconst a = 1;\nconst b = 2;\nconst c = 'it\n",
			"remove it by hand"+before,
		),
	}
	runHintCases(t, testCases)
}

func TestHint_CutVouchesOnlyForWhatItRead(t *testing.T) {
	t.Parallel()

	const (
		from  = "; could not read this file's strings from line "
		on    = " on"
		tail  = "; inside a string literal it is data, so write the language's escape instead"
		other = "remove it by hand"
	)
	testCases := []hintCase{
		outside(
			"jsx marker between a misread template and the cut",
			"a.jsx",
			"export const Key = () => <kbd>`</kbd>;\nconst price = `R$\u00A010 it's`;\n"+
				"export const Total = () => <p>{price}</p>;\n",
			other+from+"2"+on+tail,
		),
		outside(
			"go marker between a misread raw string and the cut",
			"a.go",
			"package d\n\nvar a = `\n\nvar t = `say \"hi\u00A0`\nvar u = 1\n",
			other+from+"5"+on+tail,
		),
		typographic(
			"jsx typographic between a misread template and the cut",
			"a.jsx",
			"export const Key = () => <kbd>`</kbd>;\nconst price = `wait\u2026 it's`;\n"+
				"export const Total = () => <p>{price}</p>;\n",
			"replace it with the ASCII form by hand"+from+"2"+on+tail,
		),
	}
	runHintCases(t, testCases)
}
