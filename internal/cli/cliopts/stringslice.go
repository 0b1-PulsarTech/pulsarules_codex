package cliopts

import "strings"

// stringSliceFlag binds a repeatable string flag (e.g. --target claude --target
// opencode) into a slice.
type stringSliceFlag []string

func (s *stringSliceFlag) String() string { return strings.Join(*s, ",") }

func (s *stringSliceFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}

// Options holds every CLI option, mirroring the ink_stream cmd struct format:
// one struct, filled by ParseArgs binding flags to its fields, consumed by the
// run functions. Command selects the subcommand; the rest are its options.
