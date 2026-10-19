package helperbin

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"sync"
	"time"
)

const BuildTimeout = 45 * time.Second

type Locator struct {
	files fs.FS
	root  string
	mu    sync.Mutex
	exe   string
}

func NewLocator(files fs.FS, root string) *Locator { return &Locator{files: files, root: root} }

func (locator *Locator) Exe(ctx context.Context) (string, error) {
	locator.mu.Lock()
	defer locator.mu.Unlock()
	// A rival process may prune the exe, so a remembered path is only trusted while it exists.
	if _, err := os.Stat(locator.exe); locator.exe != "" && err == nil {
		return locator.exe, nil
	}
	exe, err := locator.locate(ctx)
	if err != nil {
		return "", err
	}
	locator.exe = exe
	return exe, nil
}

func (locator *Locator) locate(ctx context.Context) (string, error) {
	files, err := fs.Sub(locator.files, locator.root)
	if err != nil {
		return "", fmt.Errorf("open embedded helper: %w", err)
	}
	dart, lookErr := exec.LookPath("dart")
	if lookErr != nil {
		return "", fmt.Errorf(
			"locate dart: no dart on PATH (fvm users need one, e.g. fvm global): %w", lookErr,
		)
	}
	base, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("locate user cache dir: %w", err)
	}
	buildCtx, cancelBuild := context.WithTimeoutCause(ctx, BuildTimeout, errBuildBound)
	defer cancelBuild()
	return ensureHelper(buildCtx, files, base, dart)
}
