package install

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

type prompter struct {
	in         *bufio.Reader
	out        io.Writer
	isTerminal bool
}

func newPrompter(in io.Reader, out io.Writer) *prompter {
	return &prompter{in: bufio.NewReader(in), out: out}
}

func (console *prompter) ask(question string) string {
	_, _ = fmt.Fprint(console.out, question)
	line, _ := console.in.ReadString('\n')
	return strings.TrimSpace(line)
}
