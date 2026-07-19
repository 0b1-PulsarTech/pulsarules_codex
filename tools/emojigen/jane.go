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

type janeUsage struct {
	byEmoji map[string]int
	byType  map[string]map[string]int
	typed   int
}

func readJaneUsage(dir string) (janeUsage, error) {
	const (
		gitMaxCount              = "--max-count=1000000"
		estimatedEmojiVocabulary = 512
	)

	allowedTypes := commitmsg.AllowedTypes()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	//nolint:gosec // dir is an operator-supplied path to a local clone.
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "log", "--format=%s", "--all", gitMaxCount)
	out, err := cmd.Output()
	if err != nil {
		return janeUsage{}, fmt.Errorf("read git log of %s: %w", dir, err)
	}

	usage := janeUsage{
		byEmoji: make(map[string]int, estimatedEmojiVocabulary),
		byType:  make(map[string]map[string]int, len(allowedTypes)),
	}
	for subject := range strings.SplitSeq(string(out), "\n") {
		usage.add(commitmsg.Parse(subject), allowedTypes)
	}
	if len(usage.byEmoji) == 0 {
		return janeUsage{}, fmt.Errorf("read git log of %s: no emoji subjects found", dir)
	}
	return usage, nil
}

func (usage *janeUsage) add(msg commitmsg.Message, allowedTypes []string) {
	const estimatedTypeVocabulary = 64

	if len(msg.Emojis) == 0 {
		return
	}
	shortcode := msg.Emojis[0]
	usage.byEmoji[shortcode]++

	if msg.Type == "" || !slices.Contains(allowedTypes, msg.Type) {
		return
	}
	usage.typed++
	bucket, ok := usage.byType[msg.Type]
	if !ok {
		bucket = make(map[string]int, estimatedTypeVocabulary)
		usage.byType[msg.Type] = bucket
	}
	bucket[shortcode]++
}
