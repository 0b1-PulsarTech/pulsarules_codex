package sandbox

import "github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"

type Request struct {
	Seq    uint64
	Source syntax.Source
	Local  []string
}

type reply struct {
	Seq  uint64
	File syntax.File
}

const protocol = 2

type hello struct{ Protocol int }
