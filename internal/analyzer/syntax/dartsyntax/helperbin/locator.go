package helperbin

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"time"
)

const BuildTimeout = 45 * time.Second

type Locator struct {
	src  source
	slot chan struct{}
	exe  string
}

func NewLocator(files fs.FS, root string) *Locator {
	return &Locator{src: source{files: files, root: root}, slot: make(chan struct{}, 1)}
}

func (locator *Locator) Exe(ctx context.Context) (string, error) {
	select {
	case locator.slot <- struct{}{}:
		defer func() { <-locator.slot }()
	case <-ctx.Done():
		return "", fmt.Errorf("wait for dart helper build: %w", context.Cause(ctx))
	}
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
	buildCtx, cancelBuild := boundBuild(ctx)
	defer cancelBuild()
	return ensureHelper(buildCtx, locator.src, base, dart)
}

func boundBuild(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeoutCause(ctx, BuildTimeout, errBuildBound)
}
