package contract

import (
	"fmt"
	"io/fs"
	"strings"
)

const contractAsset = "hooks/contract.txt"

func Session(templates fs.FS) (string, error) {
	const tailAsset = "hooks/contract-tail.txt"

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

func Subagent(templates fs.FS) (string, error) {
	// No commit tail, since a subagent never commits.
	return readAsset(templates, contractAsset)
}

func readAsset(templates fs.FS, asset string) (string, error) {
	assetBytes, err := fs.ReadFile(templates, asset)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", asset, err)
	}
	return strings.TrimSpace(string(assetBytes)), nil
}
