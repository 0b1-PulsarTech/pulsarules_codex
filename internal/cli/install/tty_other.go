//go:build !linux && !darwin

package install

func stdinIsTerminal() bool {
	// No portable isatty here without a dependency: fail closed, never interactive.
	return false
}
