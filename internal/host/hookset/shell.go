package hookset

import "strings"

func ShellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

func JoinShellList(separator rune, first, second string) string {
	// The separator is quoted wherever the shell would read it as syntax (";" ends a command).
	word := string(separator)
	if separator != ':' {
		word = `"` + word + `"`
	}
	return first + word + second
}
