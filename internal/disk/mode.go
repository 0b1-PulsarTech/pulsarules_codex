package disk

import "io/fs"

const (
	DirPrivate  fs.FileMode = 0o750
	FilePrivate fs.FileMode = 0o600
)
