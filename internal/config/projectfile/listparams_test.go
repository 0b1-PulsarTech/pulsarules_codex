package projectfile

import "testing"

func TestValidateListParams(t *testing.T) {
	t.Parallel()

	const head = "version: 1\nanalyzers:\n  big-comment:\n    params:\n      allowed_numbers: "
	runValidateCases(t, []validateCase{
		{"quoted list is accepted", head + "['0', '1']\n", ""},
		{"empty list is accepted", head + "[]\n", ""},
		{
			"unquoted numbers name analyzer, param and item",
			head + "[0, 1]\n",
			"analyzers.big-comment.params.allowed_numbers: item 1 is a number; quote it (\"0\")",
		},
		{
			"mixed list names the first offending item",
			head + "['0', 1, '2']\n",
			"allowed_numbers: item 2 is a number; quote it (\"1\")",
		},
		{"float item is a number", head + "['0', 0.5]\n", "item 2 is a number; quote it (\"0.5\")"},
		{"boolean item", head + "['0', true]\n", "item 2 is a boolean; want a string"},
		{"null item", head + "['0', ~]\n", "item 2 is empty; want a string"},
		{"nested item", head + "['0', ['1']]\n", "item 2 is a nested value; want a string"},
		{"scalar string is accepted", head + "'0,1'\n", ""},
		{"scalar number is not a list and is left to the reader", head + "5\n", ""},
	})
}
