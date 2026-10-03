package governance

import (
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/commit/message"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

func (s *Session) emojiWindow() int {
	params := pass.ParamSet(s.cfgAnalysis.Analyzers[message.ID].Params)
	hard := params.Int(message.ParamEmojiHardWindow, 0)
	// Fetch the wider window, so neither emoji rule sees fewer subjects than it counts.
	return max(hard, params.Int(message.ParamEmojiSoftWindow, 0))
}

func (s *Session) gitHistory() []pass.GitCommitEntry {
	if s.repo == nil {
		return nil
	}
	window := s.emojiWindow()
	if s.commitMsg != "" {
		// Dropping HEAD below would otherwise leave the widest window one subject short.
		window++
	}
	subjects, err := s.repo.RecentSubjects(window)
	if err != nil {
		s.logger.Warn("commit history unavailable", slog.Any("error", err))
		return nil
	}
	if len(subjects) == 0 {
		return nil
	}
	// On an amend or a rebase reword HEAD is the old commit; keeping it self-collides.
	if s.replacesHead(subjects[0]) {
		subjects = subjects[1:]
	}

	entries := make([]pass.GitCommitEntry, len(subjects))
	for i, subject := range subjects {
		// RecentSubjects is newest-first; GitHistory reads the tail, so
		// reverse it into oldest-first here.
		entries[len(subjects)-1-i] = pass.GitCommitEntry{Subject: subject}
	}
	return entries
}

func (s *Session) replacesHead(headSubject string) bool {
	if s.commitMsg == "" {
		return false
	}
	subject, _, _ := strings.Cut(s.commitMsg, "\n")
	if subject == headSubject {
		return true
	}
	// Subject alone misses an amend that also rewords it.
	return s.authorDateMatchesHead()
}

func (s *Session) authorDateMatchesHead() bool {
	// Git commit exports GIT_AUTHOR_DATE to its hooks; an amend or reword keeps HEAD's date.
	envEpoch, ok := parseGitAuthorDateEpoch(os.Getenv("GIT_AUTHOR_DATE"))
	if !ok {
		return false
	}
	headEpoch, hasHead, err := s.repo.HeadAuthorEpoch()
	if err != nil {
		s.logger.Warn("head author date unavailable", slog.Any("error", err))
		return false
	}
	if !hasHead {
		return false
	}
	// A fresh commit in HEAD's author second looks the same, so its repeat goes unseen.
	return envEpoch == headEpoch
}

func parseGitAuthorDateEpoch(raw string) (int64, bool) {
	if raw == "" {
		return 0, false
	}
	// Git accepts "@<epoch> <tz>" or a plain epoch; only the epoch field matters.
	field, _, _ := strings.Cut(raw, " ")
	epoch, err := strconv.ParseInt(strings.TrimPrefix(field, "@"), 10, 64)
	if err != nil {
		return 0, false
	}
	return epoch, true
}
