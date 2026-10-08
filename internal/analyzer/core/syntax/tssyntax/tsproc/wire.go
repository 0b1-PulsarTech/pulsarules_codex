package tsproc

import "github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax"

type Request struct {
	Seq    uint64
	Path   string
	Text   string
	IsTest bool
}

type Reply struct {
	Seq  uint64
	File syntax.File
}

const protocol = 1

type hello struct{ Protocol int }
