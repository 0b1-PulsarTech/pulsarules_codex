package core

type BlockDelim struct {
	Open  string
	Close string
}

type CommentSyntax struct {
	Line   []string
	Blocks []BlockDelim
}
