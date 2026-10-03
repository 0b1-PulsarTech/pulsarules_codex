package gosrc

import (
	"go/ast"
	"go/token"
	"regexp"
	"slices"
	"strings"
)

type QualRule struct {
	OldQual  string
	NewQual  string
	Declared func(name string) bool
}

type side struct {
	file File
	pos  int
	qual string
}

func BodiesEquivalent(oldFile, newFile File, rules []QualRule, bind Binding) bool {
	oldBody, newBody := oldFile.body, newFile.body
	oldPos, newPos := 0, 0
	for oldPos < len(oldBody) && newPos < len(newBody) {
		if sameToken(oldBody[oldPos], newBody[newPos]) ||
			sameComment(oldFile, newFile, [2]int{oldPos, newPos}, bind.Moves) {
			at := [2]int{oldPos, newPos}
			if rebindsQualifier(oldFile, newFile, at, bind) ||
				capturesBare(newFile, newPos, bind) {
				return false
			}
			oldPos++
			newPos++
			continue
		}
		oldSpan, newSpan, ok := forcedEdit(oldFile, newFile, [2]int{oldPos, newPos}, rules)
		if !ok {
			return false
		}
		oldPos += oldSpan
		newPos += newSpan
	}
	return oldPos == len(oldBody) && newPos == len(newBody)
}

func forcedEdit(
	oldFile, newFile File, at [2]int, rules []QualRule,
) (oldSpan, newSpan int, ok bool) {
	for _, rule := range rules {
		oldName, oldWidth, isOld := referenceAt(side{oldFile, at[0], rule.OldQual})
		newName, newWidth, isNew := referenceAt(side{newFile, at[1], rule.NewQual})
		if isOld && isNew && oldName == newName && ast.IsExported(oldName) &&
			rule.Declared(oldName) {
			return oldWidth, newWidth, true
		}
	}
	return 0, 0, false
}

func referenceAt(at side) (name string, width int, ok bool) {
	const qualifiedWidth = 3
	body := at.file.body
	if at.qual == "" {
		if body[at.pos].kind != token.IDENT || !at.file.unresolved[body[at.pos].off] {
			return "", 0, false
		}
		return body[at.pos].lit, 1, true
	}
	if !isQualified(body, at.pos, at.qual) || !at.file.unresolved[body[at.pos].off] {
		return "", 0, false
	}
	// A qualified reference is the three tokens q . X.
	return body[at.pos+2].lit, qualifiedWidth, true
}

func sameToken(left, right goToken) bool {
	return left.kind == right.kind && left.lit == right.lit
}

func isQualified(body []goToken, pos int, qualifier string) bool {
	if pos+2 >= len(body) || body[pos].kind != token.IDENT || body[pos].lit != qualifier {
		return false
	}
	if body[pos+1].kind != token.PERIOD || body[pos+2].kind != token.IDENT {
		return false
	}
	return pos == 0 || body[pos-1].kind != token.PERIOD
}

func rebindsQualifier(oldFile, newFile File, pos [2]int, bind Binding) bool {
	qualifier := oldFile.body[pos[0]].lit
	if !isQualified(oldFile.body, pos[0], qualifier) ||
		!isQualified(newFile.body, pos[1], qualifier) {
		return false
	}
	if !newFile.unresolved[newFile.body[pos[1]].off] {
		return false
	}
	return !bind.allows(qualifier, newFile.body[pos[1]+2].lit)
}

func sameComment(oldFile, newFile File, at [2]int, moves []MoveInfo) bool {
	left, right := oldFile.body[at[0]], newFile.body[at[1]]
	if !rewritable(oldFile, left) || !rewritable(newFile, right) {
		return false
	}
	pattern := regexp.MustCompile(`\b[A-Za-z_]\w*\.[A-Z]\w*`)
	leftRefs := pattern.FindAllString(left.lit, -1)
	rightRefs := pattern.FindAllString(right.lit, -1)
	if !slices.Equal(pattern.Split(left.lit, -1), pattern.Split(right.lit, -1)) {
		return false
	}
	for idx, ref := range leftRefs {
		if !sameReference(ref, rightRefs[idx], moves) {
			return false
		}
	}
	return true
}

func rewritable(file File, tok goToken) bool {
	if tok.kind != token.COMMENT || isDirective(tok.lit) || file.inExample(tok.off) {
		return false
	}
	text := strings.TrimLeft(
		strings.TrimPrefix(strings.TrimPrefix(tok.lit, "/*"), "//"),
		" \t\r\n",
	)
	if strings.Contains(text, "ERROR") {
		return false
	}
	return !regexp.MustCompile("^want[\\s`\"]").MatchString(text)
}

func sameReference(left, right string, moves []MoveInfo) bool {
	if left == right {
		return true
	}
	leftQual, leftName, _ := strings.Cut(left, ".")
	rightQual, rightName, _ := strings.Cut(right, ".")
	return leftName == rightName && slices.ContainsFunc(moves, func(info MoveInfo) bool {
		return info.OldPkg == leftQual && info.NewPkg == rightQual && info.Decls[leftName]
	})
}

func isDirective(comment string) bool {
	for _, prefix := range []string{"//go:", "//line ", "/*line ", "//export ", "//extern "} {
		if strings.HasPrefix(comment, prefix) {
			return true
		}
	}
	return false
}
