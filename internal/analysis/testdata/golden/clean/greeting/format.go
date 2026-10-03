package greeting

import "strings"

func (c Card) Render() string {
	parts := []string{"Dear " + c.Recipient + ",", c.Message}
	return strings.Join(parts, " ")
}

func (c Card) IsAddressed() bool {
	return c.Recipient != ""
}
