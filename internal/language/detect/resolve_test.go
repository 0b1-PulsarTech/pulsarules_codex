package detect

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
)

func TestResolve_Autodetects(t *testing.T) {
	t.Parallel()

	repoDir := t.TempDir()
	writeMarker(t, repoDir, "go.mod")

	got, err := Resolve(repoDir, nil)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if want := []string{"go"}; !slices.Equal(got.Strings(), want) {
		t.Fatalf("Resolve = %v, want %v", got.Strings(), want)
	}

	if _, err = os.Stat(filepath.Join(repoDir, ConfigFileName)); err != nil {
		t.Fatalf("expected %s to be persisted: %v", ConfigFileName, err)
	}
}

func TestResolve_TrustsPersistedConfig(t *testing.T) {
	t.Parallel()

	repoDir := t.TempDir()
	if err := SaveConfig(repoDir, NewSet(language.Go)); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	writeMarker(t, repoDir, "pubspec.yaml")

	got, err := Resolve(repoDir, nil)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if want := []string{"go"}; !slices.Equal(got.Strings(), want) {
		t.Errorf("Resolve = %v, want %v (persisted config must win over a new marker)",
			got.Strings(), want)
	}
}

func TestResolve_OverrideBeatsEverything(t *testing.T) {
	t.Parallel()

	repoDir := t.TempDir()
	if err := SaveConfig(repoDir, NewSet(language.Go)); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	writeMarker(t, repoDir, "go.mod")

	got, err := Resolve(repoDir, []string{"dart", "svelte"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if want := []string{"dart", "svelte"}; !slices.Equal(got.Strings(), want) {
		t.Fatalf("Resolve = %v, want %v", got.Strings(), want)
	}

	persisted, ok, err := LoadConfig(repoDir)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if !ok {
		t.Fatal("LoadConfig ok = false, want true (override must persist)")
	}
	if want := []string{"dart", "svelte"}; !slices.Equal(persisted.Strings(), want) {
		t.Errorf("persisted config = %v, want %v", persisted.Strings(), want)
	}
}

func TestResolve_EmptyOverrideStillWins(t *testing.T) {
	t.Parallel()

	repoDir := t.TempDir()
	writeMarker(t, repoDir, "go.mod")

	got, err := Resolve(repoDir, []string{})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !got.Empty() {
		t.Fatalf("Resolve = %v, want empty (a given flag beats detection)", got.Strings())
	}
	if _, ok, loadErr := LoadConfig(repoDir); loadErr != nil || !ok {
		t.Errorf("LoadConfig ok = %v, err = %v, want the empty override persisted", ok, loadErr)
	}
}

func TestResolve_RejectsBadOverride(t *testing.T) {
	t.Parallel()

	repoDir := t.TempDir()
	if _, err := Resolve(repoDir, []string{"cobol"}); err == nil {
		t.Fatal("Resolve with an unknown language: expected an error, got nil")
	}
	if _, err := os.Stat(filepath.Join(repoDir, ConfigFileName)); err == nil {
		t.Error("a rejected override must not persist a config file")
	}
}

func TestResolve_DoesNotPersistAnEmptyDetection(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	set, err := Resolve(dir, nil)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !set.Empty() {
		t.Fatalf("set = %v, want empty", set.Strings())
	}
	if _, statErr := os.Stat(filepath.Join(dir, ConfigFileName)); !os.IsNotExist(statErr) {
		t.Fatalf("config was written for an empty detection: %v", statErr)
	}
}
