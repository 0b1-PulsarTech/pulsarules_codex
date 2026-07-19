package governance

import "github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"

const gitHistoryLimit = 20

func (s *Session) gitHistory() []pass.GitCommitEntry {
	if s.repo == nil {
		return nil
	}
	subjects, err := s.repo.RecentSubjects(gitHistoryLimit)
	if err != nil || len(subjects) == 0 {
		return nil
	}

	entries := make([]pass.GitCommitEntry, len(subjects))
	for i, subject := range subjects {
		// RecentSubjects is newest-first; GitHistory reads the tail, so
		// reverse it into oldest-first here.
		entries[len(subjects)-1-i] = pass.GitCommitEntry{Subject: subject}
	}
	return entries
}
