package note

type Card struct {
	Text string
}

func NewCard(text string) Card {
	return Card{Text: text}
}

func (card Card) Render() string {
	if card.Text == "" {
		return "(empty)" // display only — see the style guide
	}
	return card.Text
}
