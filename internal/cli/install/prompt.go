package install

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

type Prompter struct {
	in  *bufio.Reader
	out io.Writer
}

func NewPrompter(in io.Reader, out io.Writer) *Prompter {
	return &Prompter{in: bufio.NewReader(in), out: out}
}

func (prompter *Prompter) Ask(question string) string {
	_, _ = fmt.Fprint(prompter.out, question)
	line, _ := prompter.in.ReadString('\n')
	return strings.TrimSpace(line)
}
