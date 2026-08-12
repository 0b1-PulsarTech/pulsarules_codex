package commitmsg

import (
	"strings"
)

// The commit message grammar (BNF):
//
//   message  ::= emojis? header (blank body? (blank footers?)?)?
//   emojis   ::= emoji (spaces emoji)*   (up to 3)
//   emoji    ::= ":" name ":"             (name = [a-z0-9_]+, no spaces inside)
//   header   ::= type scope? breaking? ": " description
//   scope    ::= "(" scopechars ")"
//   breaking ::= "!"
//   description ::= non-newline-chars
//   body     ::= text-paragraphs
//   footers  ::= footer (blank footer)*
//   footer   ::= footerkey (": " | " #") footervalue
//   blank    ::= "\n" "\n"
//
// Special cases (checked after parse):
//   ":ghost: Initial Commit"         → IsInitial = true (no type)
//   ":volcano: Merge ..."             → IsMerge = true (no type)
//   "[wip]" / "[WIP]" prefix in desc  → IsWIP = true

// MaxLeadingEmojis is the most leading :shortcode: emoji a commit message may
// carry. The parser consumes every leading emoji token, however many there
// are; the commit rules (see internal/analyzer/commit) reject a message
// whose Emojis exceeds this count.
const MaxLeadingEmojis = 3

// maxEmojiNameLength bounds how long a single :shortcode: name may run
// before tryEmoji gives up on it as unterminated punctuation rather than an
// emoji (the longest real shortcode is nowhere near this).
const maxEmojiNameLength = 32

// parser is the internal state machine.
type parser struct {
	runes []rune
	pos   int
}

// Parse parses a raw commit message into a Message. It is lenient:
// parse errors are recorded but the message is still returned with whatever
// fields could be extracted. The caller validates via the rules package.
func Parse(raw string) Message {
	msg := Message{Raw: raw}
	p := &parser{runes: []rune(raw)}
	p.parse(&msg)
	return msg
}

func (p *parser) parse(msg *Message) {
	p.skipLeadingWhitespace()
	p.parseEmojis(msg)
	p.skipSpaces()

	if p.isSpecialInitial(msg) {
		msg.IsInitial = true
		msg.Description = p.remainingText()
		return
	}
	if p.isSpecialMerge(msg) {
		msg.IsMerge = true
		msg.Description = p.remainingText()
		return
	}

	p.parseHeader(msg)
	if p.atEnd() {
		return
	}
	p.parseBodyAndFooters(msg)
}

// why: uncapped on purpose - MaxLeadingEmojis is a validation rule the commit
// analyzer enforces on the resulting count, not a token-count the parser
// truncates at, so an over-long run reports commit-emoji-count instead of
// the extras being misread as the type.
func (p *parser) parseEmojis(msg *Message) {
	for {
		name, ok := p.tryEmoji()
		if !ok {
			break
		}
		msg.Emojis = append(msg.Emojis, name)
		p.skipSpaces()
	}
}

// tryEmoji attempts to consume a :shortcode: at the current position. On
// success it returns the shortcode name (without colons) and advances past
// the closing colon. On failure it restores the position.
func (p *parser) tryEmoji() (string, bool) {
	start := p.pos
	if !p.consume(':') {
		p.pos = start
		return "", false
	}
	nameStart := p.pos
	for p.pos < len(p.runes) {
		r := p.runes[p.pos]
		if r == ':' {
			break
		}
		if r == ' ' || r == '\n' || p.pos-nameStart > maxEmojiNameLength {
			p.pos = start
			return "", false
		}
		if !isEmojiChar(r) {
			p.pos = start
			return "", false
		}
		p.pos++
	}
	if p.pos == nameStart || p.pos >= len(p.runes) {
		p.pos = start
		return "", false
	}
	name := string(p.runes[nameStart:p.pos])
	p.pos++
	return name, true
}

func (p *parser) parseHeader(msg *Message) {
	if !p.parseType(msg) {
		return
	}
	p.parseScope(msg)
	p.parseBreakingAndColon(msg)
	p.parseDescription(msg)
}

func (p *parser) parseBodyAndFooters(msg *Message) {
	if p.peek() == '\n' {
		p.pos++
	}
	if p.peek() != '\n' {
		return
	}
	p.pos++

	var bodyLines []string
	for p.pos < len(p.runes) {
		line := p.consumeLine()
		if isFooterLine(line) {
			footer := parseFooter(line)
			if footer.Key != "" {
				msg.Footers = append(msg.Footers, footer)
			}
			break
		}
		if line == "" {
			continue
		}
		bodyLines = append(bodyLines, line)
	}
	msg.Body = strings.TrimSpace(strings.Join(bodyLines, "\n"))

	for p.pos < len(p.runes) {
		line := p.consumeLine()
		if line == "" {
			continue
		}
		footer := parseFooter(line)
		if footer.Key != "" {
			msg.Footers = append(msg.Footers, footer)
		}
	}
}
