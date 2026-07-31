package fsperm

import "io/fs"

const (
	// File mirrors the house default for a plain, world-readable file,
	// such as a README copied out for a human to read.
	File fs.FileMode = 0o644

	// DirPrivate is this repo's directory mode: owner full access, group
	// traverse+read, others none. Used for config/data/plugin directories
	// this repo does not want world-readable.
	DirPrivate fs.FileMode = 0o750

	// FilePrivate is this repo's mode for an owner-only file that is not a
	// credential - rendered config, session markers, generated data. The
	// name says what the file actually is at the call site rather than
	// overclaiming it as secret material, which this repo stores none of.
	FilePrivate fs.FileMode = 0o600

	// FileExec is this repo's mode for a file that must stay directly
	// executable after being written, such as a copied hook script or the
	// self-copied installer binary.
	FileExec fs.FileMode = 0o755
)
