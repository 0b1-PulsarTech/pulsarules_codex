package historynarration

import (
	"slices"
	"testing"
)

func TestFindHitsPhrases(t *testing.T) {
	t.Parallel()

	for _, testCase := range slices.Concat(phraseCases(), sentenceCases(), passiveCases()) {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			runScan(t, testCase)
		})
	}
}

func phraseCases() []scanCase {
	return []scanCase{
		{"previously we", "go", "// previously we called Foo here\n", 1},
		{"block comment", "go", "/* the old implementation looped */\n", 1},
		{"sql line comment", "sql", "-- was replaced by the view\n", 1},
		{"case insensitive", "go", "// Was Replaced By the helper\n", 1},
		{"used to return", "go", "// this used to return nil\n", 1},
		{"no longer needed", "go", "// the guard is no longer needed\n", 1},
		{
			"multi-line block flags the matching line",
			"go",
			"/*\n a\n the old code did x\n*/\n",
			1,
		},
		{"was changed to", "go", "// the default was changed to 5\n", 1},
		{"previously had", "go", "// procedure previously had no way\n", 1},
		{"subject used to with any verb", "go", "// this used to swallow every error\n", 1},
		{"it used to continue", "go", "// it used to continue past errors\n", 1},
		{"never used to", "go", "// we never used to retry\n", 1},
		{"which used to", "go", "// the loop, which used to spin\n", 0},
		{"relative that used to", "go", "// the commit that used to create the row\n", 0},
		{"relative which used to", "go", "// the index which used to resolve names\n", 0},
		{"passive is used to be", "go", "// the flag is used to be explicit\n", 0},
		{"it used to be", "go", "// it used to be hardcoded\n", 1},
		{"passive used to deduplicate", "go", "// files used to deduplicate\n", 0},
		{"passive is used to resolve", "go", "// the map is used to resolve names\n", 0},
		{"noun used to continue", "go", "// count used to continue the scan\n", 0},
		{"used to after only", "go", "// this helper is only used to return the default\n", 0},
		{"used to after also", "go", "// the client is also used to call the API\n", 0},
		{"used to after mostly", "go", "// these flags are mostly used to be explicit\n", 0},
		{"used to after never", "go", "// the handle is never used to call into C\n", 0},
		{"used to after two adverbs", "go", "// it is also only used to call x\n", 0},
		{"used to after three adverbs", "go", "// it is also only just used to call x\n", 1},
		{"subject then never used to", "go", "// we never used to retry\n", 1},
		{"it used to ends the sentence", "go", "// it used to.\n", 1},
		{"they used to ends the block", "go", "// they used to\n", 1},
		{"which used to ends the sentence", "go", "// the loop which used to.\n", 0},
		{"helper has been used to", "go", "// the helper has been used to call x\n", 0},
		{"it can be used to", "go", "// it can be used to call x\n", 0},
		{"one finding per matching line", "go", "// used to be a\n// used to use b\n", 2},
		{"contracted its used to", "go", "// it's used to call the backend\n", 0},
		{"contracted thats used to", "go", "// that's used to return the default\n", 0},
		{"contracted theyre used to", "go", "// they're used to call x\n", 0},
		{"negated isnt used to", "go", "// this field isn't used to call into C\n", 0},
		{"not used to", "go", "// the handle is not used to call into C\n", 0},
		{"typically used to", "go", "// the flag is typically used to be explicit\n", 0},
		{"never ever used to still flags", "go", "// we never ever used to retry\n", 1},
		{"passive never ever used to", "go", "// it is never ever used to call x\n", 0},
		{
			"ever used to without never is purpose",
			"go",
			"// the only tool ever used to build\n",
			0,
		},
		{"lone closing paren token", "go", "// see ( foo ) used to call x\n", 1},
	}
}

func sentenceCases() []scanCase {
	return []scanCase{
		{
			"sentence end stops the look-back",
			"go",
			"// Returns the size it was. Used to be global.\n",
			1,
		},
		{
			"question end stops the look-back",
			"go",
			"// Guess which one was? Used to be global.\n",
			1,
		},
		{
			"phrase never spans a sentence end",
			"go",
			"// Callers depend on this. Used to compute the hash.\n",
			0,
		},
		{"closing paren after a sentence end", "go", "// Nothing else is.) Used to call foo\n", 1},
		{
			"closing bracket after a sentence end",
			"go",
			"// Nothing else is.] Used to call foo\n",
			1,
		},
		{"closing quote after a sentence end", "go", "// Nothing else is.\" Used to " +
			"call foo\n", 1},
		{
			"closing apostrophe after a sentence end",
			"go",
			"// Nothing else is.' Used to call foo\n",
			1,
		},
		{"exclamation end stops the look-back", "go", "// Nothing else is! Used to call foo\n", 1},
		{"curly double quote after a sentence end", "go", "// It is.\u201d Used to call foo\n", 1},
		{"curly apostrophe after a sentence end", "go", "// It is.\u2019 Used to call foo\n", 1},
		{"closing guillemet after a sentence end", "go", "// It is.\u00bb Used to call foo\n", 1},
		{"single guillemet after a sentence end", "go", "// It is.\u203a Used to call foo\n", 1},
		{"digit stays part of a word", "go", "// this used to2 be fast\n", 0},
		{"previously then a new sentence", "go", "// Same as previously. It retries.\n", 0},
	}
}

func passiveCases() []scanCase {
	cases := make([]scanCase, 0, 14)
	for _, adverb := range []string{
		"ever", "always", "usually", "currently", "generally", "commonly", "sometimes",
	} {
		cases = append(cases, scanCase{
			"adverb " + adverb, "go", "// it is " + adverb + " used to call x\n", 0,
		})
	}
	for _, auxiliary := range []string{"are", "were", "being", "was"} {
		cases = append(cases, scanCase{
			"auxiliary " + auxiliary, "go", "// they " + auxiliary + " used to call x\n", 0,
		})
	}
	for _, negated := range []string{"aren't", "wasn't", "weren't"} {
		cases = append(cases, scanCase{
			"negated " + negated, "go", "// they " + negated + " used to call x\n", 0,
		})
	}
	return cases
}
