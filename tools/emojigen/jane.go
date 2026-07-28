package main

import (
	"context"
	"fmt"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/commitmsg"
)

// janeUsage is the measured emoji vocabulary of the reference repository: how
// often each shortcode leads a subject, and how often it leads a subject of a
// given Conventional Commit type.
type janeUsage struct {
	byEmoji map[string]int
	byType  map[string]map[string]int
	typed   int
}

// gitMaxCount is required: `git log --all` piped into another process gets
// truncated at 50 subjects, which silently reduces the vocabulary to a
// fraction of the real one.
const gitMaxCount = "--max-count=1000000"

// why: shells out to git rather than internal/vcs - this scans --all
// --max-count=1000000 (a reference clone's entire history), which go-git
// would walk orders of magnitude slower, and tools/ is not a runtime
// dependency the vcs migration needs to cover.
func readJaneUsage(dir string) (janeUsage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	//nolint:gosec // dir is an operator-supplied path to a local clone.
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "log", "--format=%s", "--all", gitMaxCount)
	out, err := cmd.Output()
	if err != nil {
		return janeUsage{}, fmt.Errorf("read git log of %s: %w", dir, err)
	}

	usage := janeUsage{
		byEmoji: make(map[string]int, 512),
		byType:  make(map[string]map[string]int, len(commitmsg.AllowedTypes)),
	}
	for subject := range strings.SplitSeq(string(out), "\n") {
		usage.add(commitmsg.Parse(subject))
	}
	if len(usage.byEmoji) == 0 {
		return janeUsage{}, fmt.Errorf("read git log of %s: no emoji subjects found", dir)
	}
	return usage, nil
}

func (u *janeUsage) add(msg commitmsg.Message) {
	if len(msg.Emojis) == 0 {
		return
	}
	shortcode := msg.Emojis[0]
	u.byEmoji[shortcode]++

	if msg.Type == "" || !slices.Contains(commitmsg.AllowedTypes, msg.Type) {
		return
	}
	u.typed++
	bucket, ok := u.byType[msg.Type]
	if !ok {
		bucket = make(map[string]int, 64)
		u.byType[msg.Type] = bucket
	}
	bucket[shortcode]++
}
