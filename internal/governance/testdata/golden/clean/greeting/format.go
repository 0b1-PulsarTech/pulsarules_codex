package greeting

import "strings"

func (card Card) Render() string {
	parts := []string{"Dear " + card.Recipient + ",", card.Message}
	return strings.Join(parts, " ")
}

func (card Card) IsAddressed() bool {
	return card.Recipient != ""
}
