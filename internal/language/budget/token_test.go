package budget

import (
	"slices"
	"testing"
)

type valueCase struct {
	name string
	word string
	char byte
	want bool
}

func valueCases() []valueCase {
	const operators = "=(,[:?&|!+-*%<~^"
	fixed := []valueCase{
		{name: "start of file"},
		{name: "return", word: "return", want: true},
		{name: "typeof", word: "typeof", want: true},
		{name: "else opens a block", word: "else"},
		{name: "do opens a block", word: "do"},
		{name: "default", word: "default", want: true},
		{name: "plain name", word: "Box"},
		{name: "class keyword", word: "class"},
		{name: "close paren", char: ')'},
		{name: "semicolon", char: ';'},
		{name: "open brace", char: '{'},
		{name: "close brace", char: '}'},
		{name: "arrow", char: '>'},
		{name: "dot", char: '.'},
	}
	byOperator := make([]valueCase, 0, len(operators))
	for idx := range len(operators) {
		byOperator = append(byOperator, valueCase{
			name: "operator " + operators[idx:idx+1],
			char: operators[idx],
			want: true,
		})
	}
	return slices.Concat(fixed, byOperator)
}

func TestValueNext(t *testing.T) {
	t.Parallel()

	for _, testCase := range valueCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			scan := newScanner("")
			scan.prevWord = testCase.word
			scan.prevByte = testCase.char
			if got := scan.valueNext(); got != testCase.want {
				t.Fatalf("valueNext %v, want %v", got, testCase.want)
			}
		})
	}
}
