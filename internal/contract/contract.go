package contract

import (
	"fmt"
	"io/fs"
	"strings"
)

const (
	contractAsset = "hooks/contract.txt"
	tailAsset     = "hooks/contract-tail.txt"
)

// Session returns the full engineering contract text emitted at
// SessionStart and folded into AGENTS.md: the routing contract followed by
// the commit/verification tail.
func Session(templates fs.FS) (string, error) {
	contract, err := readAsset(templates, contractAsset)
	if err != nil {
		return "", err
	}
	tail, err := readAsset(templates, tailAsset)
	if err != nil {
		return "", err
	}
	return contract + " " + tail, nil
}

// Subagent returns the engineering contract text emitted at SubagentStart:
// the routing contract with no commit tail, since a subagent never commits.
func Subagent(templates fs.FS) (string, error) {
	return readAsset(templates, contractAsset)
}

func readAsset(templates fs.FS, asset string) (string, error) {
	assetBytes, err := fs.ReadFile(templates, asset)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", asset, err)
	}
	return strings.TrimSpace(string(assetBytes)), nil
}
