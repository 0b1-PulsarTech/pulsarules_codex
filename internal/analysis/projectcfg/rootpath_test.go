package projectcfg

import (
	"errors"
	"testing"
)

type rootCase struct{ name, root, text, want string }

func TestWithoutRoot(t *testing.T) {
	t.Parallel()

	for _, testCase := range withoutRootCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			cause := errors.New(testCase.text)
			if got := withoutRoot(cause, testCase.root); got != testCase.want {
				t.Fatalf("withoutRoot() = %q, want %q", got, testCase.want)
			}
		})
	}
}

func withoutRootCases() []rootCase {
	cases := rootBodyCases()
	cases = append(cases, rootStartCases()...)
	return append(cases, rootAdviceCases()...)
}

func rootBodyCases() []rootCase {
	return []rootCase{
		{"the root is replaced", "/r/x", "git -C /r/x log", "git -C . log"},
		{"a one byte root is kept", "/", "git -C /r/x log", "git -C /r/x log"},
		{"a bare slash is not a root path", "/", "cd / and /", "cd / and /"},
		{"an empty root is kept", "", "git -C /r/x log", "git -C /r/x log"},
		{
			"a longer sibling name is kept", "/w/cfg",
			"/w/cfg_main/.git/worktrees/cfg", "/w/cfg_main/.git/worktrees/cfg",
		},
		{"a root with a space", "/a b", "git -C /a bc and /a b log", "git -C /a bc and . log"},
		{
			"the root twice with a child", "/w/cfg",
			"cannot chdir to /w/cfg: /w/cfg/nowhere", "cannot chdir to .: ./nowhere",
		},
		{"a quoted root", "/w/cfg", "in '/w/cfg' and \"/w/cfg\"", "in '.' and \".\""},
		{"a root inside a longer path", "/w/cfg", "at /x/w/cfg here", "at /x/w/cfg here"},
		{
			"the reason after a colon", "/w/cfg",
			"fatal: not a git repository: /w/cfg", "fatal: not a git repository: .",
		},
		{"a root after an equals sign", "/w/cfg", "dir=/w/cfg x", "dir=. x"},
		{"a root in parentheses", "/w/cfg", "in (/w/cfg/x", "in (./x"},
		{"a root in backticks", "/w/cfg", "in `/w/cfg` x", "in `.` x"},
		{"a root after a tab", "/w/cfg", "in\t/w/cfg x", "in\t. x"},
		{
			"a root before a carriage return", "/w/cfg",
			"git -C /w/cfg x\r\nfatal: at /w/cfg\r\n", "git -C . x\r\nfatal: at .\r\n",
		},
		{"a root at the start of the text", "/w/cfg", "/w/cfg is bad", ". is bad"},
		{"a root at the end of the text", "/w/cfg", "bad /w/cfg", "bad ."},
	}
}

func rootStartCases() []rootCase {
	return []rootCase{
		{"a digit before the root", "/w/cfg", "/x1/w/cfg here", "/x1/w/cfg here"},
		{"a capital before the root", "/w/cfg", "/X/w/cfg here", "/X/w/cfg here"},
		{"an underscore before the root", "/w/cfg", "/x_/w/cfg here", "/x_/w/cfg here"},
		{"a dash before the root", "/w/cfg", "/x-/w/cfg here", "/x-/w/cfg here"},
		{"a dot before the root", "/w/cfg", "/x./w/cfg here", "/x./w/cfg here"},
		{"a plus before the root", "/w/cfg", "/x+/w/cfg/y", "/x+/w/cfg/y"},
		{"a multibyte name before the root", "/w/cfg", "/caf\u00e9/w/cfg/x", "/caf\u00e9/w/cfg/x"},
		{"a root after a colon", "/w/cfg", "in :/w/cfg x", "in :. x"},
		{"a root after a comma is kept", "/w/cfg", "in ,/w/cfg x", "in ,/w/cfg x"},
		{"a root after a slash is kept", "/w/cfg", "at //w/cfg x", "at //w/cfg x"},
	}
}

func rootAdviceCases() []rootCase {
	const command = "git config --global --add safe.directory /w/cfg"
	return []rootCase{
		{
			"a root named like the advice", "/w/safe.directory/cfg",
			"git -C /w/safe.directory/cfg log", "git -C . log",
		},
		{
			"a root ending in the advice name", "/w/safe.directory",
			"at /w/safe.directory and /w/safe.directory x", "at . and . x",
		},
		{
			"the root before the advice mark", "/w/cfg",
			"at /w/cfg safe.directory /w/cfg", "at . safe.directory /w/cfg",
		},
		{
			"the safe.directory command is kept", "/w/cfg",
			"fatal: detected dubious ownership in repository at '/w/cfg'\n" + command,
			"fatal: detected dubious ownership in repository at '.'\n" + command,
		},
		{
			"the line after a tabbed command is replaced", "/w/cfg",
			"\t" + command + "\nfatal: at /w/cfg", "\t" + command + "\nfatal: at .",
		},
		{
			"a mark glued to the root keeps the rest", "/w/cfg",
			"at /w/cfgsafe.directory /w/cfg", "at /w/cfgsafe.directory /w/cfg",
		},
		{
			"a line that starts with the mark is kept", "/w/cfg",
			"safe.directory /w/cfg", "safe.directory /w/cfg",
		},
	}
}

func TestReplaceRoot(t *testing.T) {
	t.Parallel()

	testCases := []struct{ name, text, root, want string }{
		{"an empty root is kept", "git -C /r/x log", "", "git -C /r/x log"},
		{"an empty text is kept", "", "/w/cfg", ""},
		{"the root is replaced", "git -C /w/cfg log", "/w/cfg", "git -C . log"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := replaceRoot(testCase.text, testCase.root); got != testCase.want {
				t.Fatalf("replaceRoot() = %q, want %q", got, testCase.want)
			}
		})
	}
}
