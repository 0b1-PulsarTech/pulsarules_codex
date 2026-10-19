package sandbox

import (
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"strconv"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

const (
	ChildArg     = "__tsproc_child"
	exitMemory   = 75
	exitProtocol = 76
	MiB          = 1 << 20
)

var errNoCeiling = errors.New("missing memory ceiling")

type Handler func(Request) *syntax.File

func Child(args []string, handle Handler, exit func(int)) bool {
	if len(args) < 2 || args[1] != ChildArg {
		return false
	}
	ceiling, err := parseCeiling(args[2:])
	if err != nil {
		exit(exitProtocol)
		return true
	}
	// The soft limit keeps garbage from reading as live memory against the ceiling.
	//nolint:gosec // The ceiling was parsed with 63 bits, so it fits.
	debug.SetMemoryLimit(int64(ceiling / 2))
	exit(serve(os.Stdin, os.Stdout, handle, serveLimits{Ceiling: ceiling, Exit: exit}))
	return true
}

func parseCeiling(rest []string) (uint64, error) {
	if len(rest) == 0 {
		return 0, errNoCeiling
	}
	ceiling, err := strconv.ParseUint(rest[0], 10, 63)
	if err != nil {
		return 0, fmt.Errorf("parse ceiling: %w", err)
	}
	return ceiling, nil
}
