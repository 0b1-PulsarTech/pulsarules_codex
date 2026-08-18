package greeting

// Card holds a short greeting addressed to a recipient.
type Card struct {
	Recipient string
	Message   string
}

// NewCard creates a Card addressed to recipient with the given message.
func NewCard(recipient, message string) Card {
	return Card{Recipient: recipient, Message: message}
}
