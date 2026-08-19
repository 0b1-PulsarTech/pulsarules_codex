package knowledge

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
)

// standardsRoot is the dir both fingerprint sources are rooted at, so an
// embedded digest and an on-disk one describe the same subtree.
const standardsRoot = "standards"

// Fingerprint digests the standards this binary EMBEDS.
//
// why: the digest travels INSIDE the binary, so it needs no record file beside
// it - which removes the whole class of "record missing versus unreadable"
// ambiguity a sidecar file would have to distinguish.
func Fingerprint() (string, error) {
	return fingerprintFS(standardsFS)
}

// FingerprintDir digests the standards tree under root on disk (root/standards),
// producing a value comparable to Fingerprint.
func FingerprintDir(root string) (string, error) {
	return fingerprintFS(os.DirFS(root))
}

// fingerprintFS digests every file under standardsRoot in lexical walk order.
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
			// why: a FIXED-SIZE digest per file keeps the stream injective. Feeding
			// the path plus raw content would let two files hash the same as one
			// whose body happens to spell the next file's path.
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
