package disk

import "io/fs"

const (
	ModeFile        fs.FileMode = 0o644
	ModeDirPrivate  fs.FileMode = 0o750
	ModeFilePrivate fs.FileMode = 0o600
	ModeFileExec    fs.FileMode = 0o755
)
