package main

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/commitmsg"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/process"
)

type commitUsage struct {
	byEmoji map[string]int
	byType  map[string]map[string]int
	typed   int
}

func readUsage(dir string) (commitUsage, error) {
	const (
		gitMaxCount              = "--max-count=1000000"
		gitLogTimeout            = 2 * time.Minute
		estimatedEmojiVocabulary = 512
	)

	allowedTypes := commitmsg.AllowedTypes()
	// Calls git directly, not internal/vcs: tools/ is no runtime dependency that package
	// must cover. A large history walked with --all can take a while, hence the timeout.
	result, err := process.Run(context.Background(), process.Command{
		Name:    "git",
		Args:    []string{"log", "--format=%s", "--all", gitMaxCount},
		Dir:     dir,
		Timeout: gitLogTimeout,
	})
	if err != nil {
		return commitUsage{}, fmt.Errorf("read git log of %s: %w", dir, err)
	}

	usage := commitUsage{
		byEmoji: make(map[string]int, estimatedEmojiVocabulary),
		byType:  make(map[string]map[string]int, len(allowedTypes)),
	}
	for subject := range strings.SplitSeq(result.Stdout, "\n") {
		usage.add(commitmsg.Parse(subject), allowedTypes)
	}
	if len(usage.byEmoji) == 0 {
		return commitUsage{}, fmt.Errorf("read git log of %s: no emoji subjects found", dir)
	}
	return usage, nil
}

func (usage *commitUsage) add(msg commitmsg.Message, allowedTypes []string) {
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
