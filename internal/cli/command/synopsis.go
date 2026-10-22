package command

import (
	"errors"
	"flag"
	"fmt"
	"strings"
)

type Term struct {
	names    []string
	required bool
	literal  string
}

func Opt(names ...string) Term { return Term{names: names} }

func Req(names ...string) Term { return Term{names: names, required: true} }

func Lit(text string) Term { return Term{literal: text} }

func Synopsis(fs *flag.FlagSet, terms []Term) (string, error) {
	words := make([]string, 0, len(terms))
	for _, term := range terms {
		word, err := term.render(fs)
		if err != nil {
			return "", err
		}
		words = append(words, word)
	}
	return strings.Join(words, " "), nil
}

func (term Term) render(fs *flag.FlagSet) (string, error) {
	if term.literal != "" {
		return term.literal, nil
	}
	pieces := make([]string, 0, len(term.names))
	for _, name := range term.names {
		piece, err := flagPiece(fs, name)
		if err != nil {
			return "", err
		}
		pieces = append(pieces, piece)
	}
	joined := strings.Join(pieces, " | ")
	switch {
	case !term.required:
		return "[" + joined + "]", nil
	case len(pieces) > 1:
		return "(" + joined + ")", nil
	}
	return joined, nil
}

func flagPiece(fs *flag.FlagSet, name string) (string, error) {
	found := fs.Lookup(name)
	if found == nil {
		return "", fmt.Errorf("synopsis names --%s, which the command does not bind", name)
	}
	if boolean, ok := found.Value.(interface{ IsBoolFlag() bool }); ok && boolean.IsBoolFlag() {
		return "--" + name, nil
	}
	if !strings.Contains(found.Usage, "`") {
		return "", errors.New("flag --" + name + " has no `metavar` in its usage")
	}
	metavar, _ := flag.UnquoteUsage(found)
	return "--" + name + " " + metavar, nil
}
