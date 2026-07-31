package analysis

import (
	"os"
	"strconv"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

// gitHistoryLimit covers the widest emoji repetition window the commit
// rules look back over.
const gitHistoryLimit = 20

// gitHistory returns the most recent commit subjects as GitCommitEntry
// values, OLDEST FIRST, for the emoji repetition windows. When the commit
// being written REPLACES HEAD (an amend or a rebase reword), HEAD is
// dropped: it is the old version of this very commit, and leaving it in
// makes every reword collide with itself.
func (s *Session) gitHistory() []core.GitCommitEntry {
	if s.repo == nil {
		return nil
	}
	subjects, err := s.repo.RecentSubjects(gitHistoryLimit)
	if err != nil || len(subjects) == 0 {
		return nil
	}
	if s.replacesHead(subjects[0]) {
		subjects = subjects[1:]
	}

	entries := make([]core.GitCommitEntry, len(subjects))
	for i, subject := range subjects {
		// RecentSubjects is newest-first; GitHistory reads the tail, so
		// reverse it into oldest-first here.
		entries[len(subjects)-1-i] = core.GitCommitEntry{Subject: subject}
	}
	return entries
}

// replacesHead reports whether the commit message being validated replaces
// HEAD (an amend or a reword) rather than following it: either it carries
// the same subject line HEAD already has, or GIT_AUTHOR_DATE (set by git
// during an amend/rebase) matches HEAD's own author date - the subject
// alone misses an amend that also changes the subject.
func (s *Session) replacesHead(headSubject string) bool {
	if s.commitMsg == "" {
		return false
	}
	subject, _, _ := strings.Cut(s.commitMsg, "\n")
	if subject == headSubject {
		return true
	}
	return s.authorDateMatchesHead()
}

// authorDateMatchesHead reports whether the GIT_AUTHOR_DATE environment
// variable git sets during an amend/rebase equals HEAD's own author date.
func (s *Session) authorDateMatchesHead() bool {
	envEpoch, ok := parseGitAuthorDateEpoch(os.Getenv("GIT_AUTHOR_DATE"))
	if !ok {
		return false
	}
	headEpoch, hasHead, err := s.repo.HeadAuthorEpoch()
	if err != nil || !hasHead {
		return false
	}
	return envEpoch == headEpoch
}

// why: GIT_AUTHOR_DATE takes either "@<epoch> <tz>" or a plain epoch; only
// the leading epoch field matters for an equality check against HEAD's own.
func parseGitAuthorDateEpoch(raw string) (int64, bool) {
	if raw == "" {
		return 0, false
	}
	field, _, _ := strings.Cut(raw, " ")
	epoch, err := strconv.ParseInt(strings.TrimPrefix(field, "@"), 10, 64)
	if err != nil {
		return 0, false
	}
	return epoch, true
}
