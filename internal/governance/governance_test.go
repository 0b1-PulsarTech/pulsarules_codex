package governance

import (
	"context"
	"os/exec"
	"testing"
)

func gitInitRepo(t *testing.T, dir string) {
	t.Helper()
	ctx := context.Background()
	if out, err := exec.CommandContext(ctx, "git", "-C", dir, "init").
		CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	if err := exec.CommandContext(ctx, "git", "-C", dir, "config", "user.email", "test@test.com").
		Run(); err != nil {
		t.Fatalf("git config: %v", err)
	}
	if err := exec.CommandContext(ctx, "git", "-C", dir, "config", "user.name", "Test").
		Run(); err != nil {
		t.Fatalf("git config: %v", err)
	}
}
