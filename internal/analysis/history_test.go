package analysis

import "testing"

func TestSessionGitHistory_NilRepo(t *testing.T) {
	t.Parallel()

	sess := NewSession(nil, "anything", nil, nil)
	if entries := sess.gitHistory(); entries != nil {
		t.Fatalf("gitHistory() = %+v, want nil for a nil repo", entries)
	}
}
