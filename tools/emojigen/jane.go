package main

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/commitmsg"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/execx"
)

type janeUsage struct {
	byEmoji map[string]int
	byType  map[string]map[string]int
	typed   int
}

func readJaneUsage(dir string) (janeUsage, error) {
	const (
		gitMaxCount              = "--max-count=1000000"
		janeGitTimeout           = 2 * time.Minute
		estimatedEmojiVocabulary = 512
	)

	allowedTypes := commitmsg.AllowedTypes()
	// Calls git directly, not internal/vcs: tools/ is no runtime dependency that package
	// must cover. A large history walked with --all can take a while, hence the timeout.
	result, err := execx.Run(context.Background(), execx.Command{
		Name:    "git",
		Args:    []string{"log", "--format=%s", "--all", gitMaxCount},
		Dir:     dir,
		Timeout: janeGitTimeout,
	})
	if err != nil {
		return janeUsage{}, fmt.Errorf("read git log of %s: %w", dir, err)
	}

	usage := janeUsage{
		byEmoji: make(map[string]int, estimatedEmojiVocabulary),
		byType:  make(map[string]map[string]int, len(allowedTypes)),
	}
	for subject := range strings.SplitSeq(result.Stdout, "\n") {
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
