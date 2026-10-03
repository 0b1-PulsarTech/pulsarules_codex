package greeting

type Card struct {
	Recipient string
	Message   string
}

func NewCard(recipient, message string) Card {
	return Card{Recipient: recipient, Message: message}
}
