//go:build !linux && !darwin

package install

// why: no portable ioctl-based isatty on this GOOS without a new dependency;
// failing closed (never interactive) matches the safe default - install
// requires an explicit --all/--skills/--router-only instead of guessing.
func stdinIsTerminal() bool { return false }
