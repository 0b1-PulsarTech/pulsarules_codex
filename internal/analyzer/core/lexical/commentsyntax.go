package lexical

type BlockDelim struct {
	Open   string
	Close  string
	Nested bool
}

type CommentSyntax struct {
	Line   []string
	Blocks []BlockDelim
}
