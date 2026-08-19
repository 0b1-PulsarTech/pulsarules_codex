package knowledge

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
)

const standardsRoot = "standards"

func Fingerprint() (string, error) {
	return fingerprintFS(standardsFS)
}

func FingerprintDir(root string) (string, error) {
	return fingerprintFS(os.DirFS(root))
}

func fingerprintFS(fsys fs.FS) (string, error) {
	digest := sha256.New()
	err := fs.WalkDir(
		fsys,
		standardsRoot,
		func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			content, readErr := fs.ReadFile(fsys, path)
			if readErr != nil {
				return fmt.Errorf("read %q: %w", path, readErr)
			}
			// A fixed-size digest per file keeps the stream injective; raw content could
			// spell the next file's path.
			fileDigest := sha256.Sum256(content)
			_, _ = fmt.Fprintf(digest, "%s\x00%x\n", path, fileDigest)
			return nil
		},
	)
	if err != nil {
		return "", fmt.Errorf("fingerprint standards: %w", err)
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}
