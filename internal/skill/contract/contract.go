package contract

import (
	"fmt"
	"io/fs"
	"strings"
)

const contractAsset = "hooks/contract.txt"

func Session(templates fs.FS) (string, error) {
	assetBytes, err := fs.ReadFile(templates, contractAsset)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", contractAsset, err)
	}
	return strings.TrimSpace(string(assetBytes)), nil
}
