package greeting

import "strings"

// Render returns the card as a single line: "Dear <recipient>, <message>".
func (c Card) Render() string {
	parts := []string{"Dear " + c.Recipient + ",", c.Message}
	return strings.Join(parts, " ")
}

// IsAddressed reports whether the card names a recipient.
func (c Card) IsAddressed() bool {
	return c.Recipient != ""
}
