package identifier

import (
	"fmt"
	"go/token"
)

type fileChecker struct {
	index *fileIndex
	emit  func(pos token.Pos, msg, suggestion string)
}

func (checker fileChecker) run() {
	for _, bound := range checker.index.checked {
		checker.check(bound.name, bound.pos)
	}
}

func (checker fileChecker) check(name string, pos token.Pos) {
	if name == "_" || name == "" {
		return
	}

	if checker.isSequential(name) {
		checker.emit(
			pos,
			fmt.Sprintf("numbered name %q suggests sequential naming", name),
			"use a descriptive name instead of a number suffix",
		)
	}

	if checkHungarian(name) {
		checker.emit(
			pos,
			fmt.Sprintf("name %q appears to use Hungarian notation", name),
			"use plain descriptive names without type prefixes",
		)
	}

	if checker.isNoise(name, pos) {
		checker.emit(
			pos,
			fmt.Sprintf("name %q contains a noise word", name),
			"use a more specific descriptive name",
		)
	}
}

func (checker fileChecker) isSequential(name string) bool {
	stem, value, ok := numberedStem(name)
	if !ok || !isCounterValue(value) {
		return false
	}
	// A lone digit proves nothing (sha256, oauth2); a sibling makes it a sequence.
	return checker.index.hasNumberedSibling(name, stem)
}

func (checker fileChecker) isNoise(name string, pos token.Pos) bool {
	// A binding that repeats the type it was built from is exempt, per binding site.
	return checkNoiseWord(name) && !checker.index.typeDerived[pos]
}
