package bootstrap

// Options carries the CLI-resolved inputs the composition root needs to wire
// the injector. It stays independent of cmd/pulsarules_cli's own Options
// struct so this package never imports cmd.
type Options struct {
	// Root selects the knowledge source: empty reads the embedded snapshot,
	// non-empty reads <Root>/knowledge from disk (dev mode).
	Root string
	// ProjectDir resolves the vcs.Repository factory. It may not point at a
	// git repository; the factory's error surfaces to whichever command
	// handler asks for the repository.
	ProjectDir string
	// LogLevel gates the hook-execution logger; empty disables logging.
	LogLevel string
	// LogPath is the hook-execution logger's file path; empty disables
	// logging even when LogLevel is set, since obs holds no host-layout
	// default of its own to fall back to.
	LogPath string
}
