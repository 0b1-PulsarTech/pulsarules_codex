package note

type Card struct {
	Text string
}

func NewCard(text string) Card {
	return Card{Text: text}
}

func (c Card) Render() string {
	if c.Text == "" {
		return "(empty)" // display only — see the style guide
	}
	return c.Text
}
