package governance

import (
	"os"
	"strconv"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

const gitHistoryLimit = 20

func (s *Session) gitHistory() []pass.GitCommitEntry {
	if s.repo == nil {
		return nil
	}
	subjects, err := s.repo.RecentSubjects(gitHistoryLimit)
	if err != nil || len(subjects) == 0 {
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
	if err != nil || !hasHead {
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
