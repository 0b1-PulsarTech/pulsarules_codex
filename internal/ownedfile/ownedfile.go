package ownedfile

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
)

const Marker = "Installed by pulsarules_cli"

func Check(path string) (exists, ours bool, err error) {
	var content []byte
	content, exists, err = readExisting(path)
	return exists, exists && bytes.Contains(content, []byte(Marker)), err
}

func readExisting(path string) (content []byte, exists bool, err error) {
	content, err = os.ReadFile(path) //nolint:gosec // path is under the caller's directory.
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("read %q: %w", path, err)
	}
	return content, true, nil
}
