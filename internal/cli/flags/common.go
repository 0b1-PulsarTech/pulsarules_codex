package flags

import "flag"

func BindCommon(fs *flag.FlagSet, root, logLevel *string, env Environment) {
	fs.StringVar(root, "root", "", "repository root `DIR` for dev mode (defaults to embedded)")
	fs.StringVar(
		logLevel,
		"log-level",
		env.LogLevel,
		"hook execution log level: debug|info|warn|error (empty disables logging)",
	)
}
