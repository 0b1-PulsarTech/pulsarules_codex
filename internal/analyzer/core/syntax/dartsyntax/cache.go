package dartsyntax

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsperm"
)

const failureWindow = 10 * time.Minute

type failureMarker struct {
	Key     string    `json:"key"`
	At      time.Time `json:"at"`
	Message string    `json:"message"`
}

func cacheRoot(base string) string {
	return filepath.Join(base, "pulsarules", "dart-syntax")
}

func cacheDir(base, version, hash string) string {
	return filepath.Join(cacheRoot(base), keyPrefix(version)+hash)
}

func keyPrefix(version string) string {
	return version + "-" + runtime.GOOS + "-" + runtime.GOARCH + "-"
}

func exeExists(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, exeName()))
	return err == nil
}

func publish(work, dir string) error {
	err := os.Rename(work, dir)
	if err == nil || exeExists(dir) {
		return nil
	}
	// A directory without the exe is a crashed or hand-edited leftover.
	if removeErr := os.RemoveAll(dir); removeErr != nil {
		return fmt.Errorf("remove broken helper dir: %w", removeErr)
	}
	if renameErr := os.Rename(work, dir); renameErr != nil && !exeExists(dir) {
		return fmt.Errorf("publish helper: %w", renameErr)
	}
	return nil
}

func markerPath(dir string) string {
	return filepath.Join(filepath.Dir(dir), "failed-"+filepath.Base(dir)+".json")
}

func recordFailure(dir string, cause error) error {
	marker := failureMarker{Key: filepath.Base(dir), At: time.Now(), Message: cause.Error()}
	payload, err := json.Marshal(marker)
	if err != nil {
		return fmt.Errorf("encode failure marker: %w", err)
	}
	if mkdirErr := os.MkdirAll(filepath.Dir(dir), fsperm.DirPrivate); mkdirErr != nil {
		return fmt.Errorf("create helper cache: %w", mkdirErr)
	}
	if writeErr := os.WriteFile(markerPath(dir), payload, fsperm.FilePrivate); writeErr != nil {
		return fmt.Errorf("write failure marker: %w", writeErr)
	}
	return nil
}

func loadMarker(dir string) (failureMarker, bool) {
	var marker failureMarker
	payload, err := os.ReadFile(markerPath(dir))
	// An unreadable marker only costs one rebuild attempt, so it counts as absent.
	if err != nil || json.Unmarshal(payload, &marker) != nil {
		return marker, false
	}
	return marker, marker.Key == filepath.Base(dir)
}

func recentFailure(dir string) error {
	marker, ok := loadMarker(dir)
	age := time.Since(marker.At)
	if !ok || age < 0 || age >= failureWindow {
		return nil
	}
	return fmt.Errorf(
		"dart helper build failed %s ago, not retrying for %s: %s",
		age.Round(time.Second), failureWindow, marker.Message,
	)
}

func clearFailure(dir string) error {
	if err := os.Remove(markerPath(dir)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("remove failure marker: %w", err)
	}
	return nil
}
