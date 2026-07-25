package fsperm

import "io/fs"

const (
	File        fs.FileMode = 0o644
	DirPrivate  fs.FileMode = 0o750
	FilePrivate fs.FileMode = 0o600
	FileExec    fs.FileMode = 0o755
)
