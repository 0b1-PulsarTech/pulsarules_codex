package filecount

import "github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/gomod"

type srcInfo struct {
	pkg string
	ent bool
}

func inspectSource(file gomod.File) srcInfo {
	if file.Syntax == nil {
		return srcInfo{}
	}
	return srcInfo{
		pkg: file.Syntax.Name.Name,
		ent: inspectEnt(file.Syntax),
	}
}
