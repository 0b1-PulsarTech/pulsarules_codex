package helperbin

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/process"
)

var errBuildBound = errors.New("dart helper build exceeded its time bound")

func ensureHelper(ctx context.Context, files fs.FS, base, dart string) (string, error) {
	version, err := sdkVersion(ctx, dart)
	if err != nil {
		return "", err
	}
	hash, err := helperHash(files)
	if err != nil {
		return "", err
	}
	dir := newHelperDir(base, version, hash)
	if dir.exists() {
		dir.touch()
		return dir.exe(), nil
	}
	if recent := dir.recentFailure(); recent != nil {
		return "", recent
	}
	if buildErr := buildHelper(ctx, files, dart, dir); buildErr != nil {
		return "", rememberFailure(ctx, dir, buildErr)
	}
	// Tidying is hygiene; a leftover file must not fail a good build.
	_ = errors.Join(dir.clearFailure(), dir.prune())
	return dir.exe(), nil
}

func rememberFailure(ctx context.Context, dir helperDir, buildErr error) error {
	if ctx.Err() != nil && !errors.Is(context.Cause(ctx), errBuildBound) {
		return buildErr
	}
	if markErr := dir.recordFailure(buildErr); markErr != nil {
		return errors.Join(buildErr, markErr)
	}
	return buildErr
}

func buildHelper(ctx context.Context, files fs.FS, dart string, dir helperDir) error {
	if err := os.MkdirAll(dir.parent(), disk.DirPrivate); err != nil {
		return fmt.Errorf("create helper cache: %w", err)
	}
	work, err := os.MkdirTemp(dir.parent(), "build-*")
	if err != nil {
		return fmt.Errorf("create helper build dir: %w", err)
	}
	buildErr := compileAndPublish(ctx, files, toolchain{dir: work, dart: dart}, dir)
	if cleanErr := os.RemoveAll(work); cleanErr != nil {
		return errors.Join(buildErr, fmt.Errorf("remove helper build dir: %w", cleanErr))
	}
	return buildErr
}

func compileAndPublish(ctx context.Context, files fs.FS, tool toolchain, dir helperDir) error {
	if err := materialise(files, tool.dir); err != nil {
		return err
	}
	if err := pubGet(ctx, tool); err != nil {
		return err
	}
	if err := runStep(ctx, tool, "compile exe", "compile", "exe",
		filepath.Join("bin", "summary.dart"), "-o", exeName()); err != nil {
		return err
	}
	// A concurrent build may have won the rename; its exe is equivalent.
	return dir.publish(tool.dir)
}

func pubGet(ctx context.Context, tool toolchain) error {
	offlineErr := runStep(ctx, tool, "pub get offline", "pub", "get", "--offline")
	if offlineErr == nil {
		return nil
	}
	if ctx.Err() != nil {
		return offlineErr
	}
	onlineErr := runStep(ctx, tool, "pub get", "pub", "get")
	if onlineErr == nil {
		return nil
	}
	return errors.Join(offlineErr, onlineErr)
}

func dartArgs(args []string) []string {
	return append([]string{"--suppress-analytics"}, args...)
}

type toolchain struct {
	dir  string
	dart string
}

func runStep(ctx context.Context, tool toolchain, step string, args ...string) error {
	command := process.Command{Name: tool.dart, Args: dartArgs(args), Dir: tool.dir}
	result, err := process.Run(ctx, command)
	if err != nil {
		return StepError(step, result, err)
	}
	return nil
}
