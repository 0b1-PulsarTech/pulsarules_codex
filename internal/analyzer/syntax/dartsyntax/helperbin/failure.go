package helperbin

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
)

const failureWindow = 10 * time.Minute

type failureMarker struct {
	Key     string    `json:"key"`
	At      time.Time `json:"at"`
	Message string    `json:"message"`
}

func (dir helperDir) markerPath() string {
	return filepath.Join(dir.parent(), "failed-"+dir.name()+".json")
}

func (dir helperDir) recordFailure(cause error) error {
	marker := failureMarker{Key: dir.name(), At: time.Now(), Message: cause.Error()}
	payload, err := json.Marshal(marker)
	if err != nil {
		return fmt.Errorf("encode failure marker: %w", err)
	}
	if mkdirErr := os.MkdirAll(dir.parent(), disk.ModeDirPrivate); mkdirErr != nil {
		return fmt.Errorf("create helper cache: %w", mkdirErr)
	}
	if writeErr := os.WriteFile(dir.markerPath(), payload, disk.ModeFilePrivate); writeErr != nil {
		return fmt.Errorf("write failure marker: %w", writeErr)
	}
	return nil
}

func (dir helperDir) loadMarker() (failureMarker, bool) {
	var marker failureMarker
	//nolint:gosec // The marker lives inside the helper cache this process owns.
	payload, err := os.ReadFile(dir.markerPath())
	// An unreadable marker only costs one rebuild attempt, so it counts as absent.
	if err != nil || json.Unmarshal(payload, &marker) != nil {
		return marker, false
	}
	return marker, marker.Key == dir.name()
}

func (dir helperDir) recentFailure() error {
	marker, ok := dir.loadMarker()
	age := time.Since(marker.At)
	if !ok || age < 0 || age >= failureWindow {
		return nil
	}
	return fmt.Errorf(
		"dart helper build failed %s ago, not retrying for %s: %s",
		age.Round(time.Second), failureWindow, marker.Message,
	)
}

func (dir helperDir) clearFailure() error {
	if err := os.Remove(dir.markerPath()); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("remove failure marker: %w", err)
	}
	return nil
}
