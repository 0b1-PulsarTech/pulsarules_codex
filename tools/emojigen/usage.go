package main

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/commitmsg"
)

type commitUsage struct {
	byEmoji map[string]int
}

func readUsage(dir string) (commitUsage, error) {
	const (
		gitMaxCount              = "--max-count=1000000"
		estimatedEmojiVocabulary = 512
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	//nolint:gosec // dir is an operator-supplied path to a local clone.
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "log", "--format=%s", "--all", gitMaxCount)
	out, err := cmd.Output()
	if err != nil {
		return commitUsage{}, fmt.Errorf("read git log of %s: %w", dir, err)
	}

	usage := commitUsage{
		byEmoji: make(map[string]int, estimatedEmojiVocabulary),
	}
	for subject := range strings.SplitSeq(string(out), "\n") {
		usage.add(commitmsg.Parse(subject))
	}
	if len(usage.byEmoji) == 0 {
		return commitUsage{}, fmt.Errorf("read git log of %s: no emoji subjects found", dir)
	}
	return usage, nil
}

func (usage *commitUsage) add(msg commitmsg.Message) {
	if len(msg.Emojis) == 0 {
		return
	}
	shortcode := msg.Emojis[0]
	usage.byEmoji[shortcode]++
}
