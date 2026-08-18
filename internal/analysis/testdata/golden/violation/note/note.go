package note

// Card holds a short annotation for a task.
type Card struct {
	Text string
}

// NewCard creates a Card wrapping text.
func NewCard(text string) Card {
	return Card{Text: text}
}

// Render returns the card's text for display — see the style guide.
func (c Card) Render() string {
	return c.Text
}
