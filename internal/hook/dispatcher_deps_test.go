package hook

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/dartsyntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

func TestDispatcher_SkillInstalled(t *testing.T) {
	t.Parallel()

	projectDir, globalDir := t.TempDir(), t.TempDir()
	installSkillFixture(t, projectDir, "go-style")
	installSkillFixture(t, globalDir, "database-persistence")
	sep := string(os.PathListSeparator)
	testCases := []struct {
		name      string
		skillsDir string
		id        string
		want      bool
	}{
		{"installed", projectDir, "go-style", true},
		{"not installed", projectDir, "not-installed", false},
		{"no skills dir", "", "go-style", false},
		{
			"only in the global dir",
			projectDir + sep + globalDir, "database-persistence", true,
		},
		{
			"in neither dir",
			projectDir + sep + globalDir, "not-installed", false,
		},
		{
			"missing first dir does not hide the second",
			filepath.Join(projectDir, "absent") + sep + globalDir,
			"database-persistence", true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			disp := NewDispatcher(Deps{SkillsDir: testCase.skillsDir})
			if got := disp.skillInstalled(testCase.id); got != testCase.want {
				t.Errorf("skillInstalled(%q) = %v, want %v", testCase.id, got, testCase.want)
			}
		})
	}
}

func TestDispatcher_MissingProjectDirOnce(t *testing.T) {
	t.Parallel()

	var logBuf, errBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))
	disp, _ := dispatchCapture(Deps{Logger: logger, ErrOut: &errBuf})
	id := uniqueSessionID(t)
	payload := fmt.Appendf(nil, `{"session_id":%q}`, id)

	_ = disp.Dispatch("session-start", payload)
	_ = disp.Dispatch("user-prompt", payload)

	if got := strings.Count(errBuf.String(), "reinstall"); got != 1 {
		t.Errorf(
			"expected exactly one stderr reinstall warning across two dispatches, got %d in:\n%s",
			got, errBuf.String(),
		)
	}
	if got := strings.Count(logBuf.String(), "reinstall"); got != 1 {
		t.Errorf(
			"expected exactly one logged reinstall warning across two dispatches, got %d in:\n%s",
			got, logBuf.String(),
		)
	}
}

func TestDispatcher_MissingProjectDirWarning(t *testing.T) {
	t.Parallel()

	var logBuf, errBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))
	disp, _ := dispatchCapture(
		Deps{Logger: logger, ErrOut: &errBuf, ProjectDir: t.TempDir()},
	)

	_ = disp.Dispatch("session-start", newSessionPayload(t))

	if strings.Contains(errBuf.String(), "reinstall") {
		t.Errorf(
			"unexpected stderr reinstall warning with a resolved project dir:\n%s",
			errBuf.String(),
		)
	}
	if strings.Contains(logBuf.String(), "reinstall") {
		t.Errorf(
			"unexpected logged reinstall warning with a resolved project dir:\n%s",
			logBuf.String(),
		)
	}
}

func TestDispatcher_WarnsToStderr(t *testing.T) {
	t.Parallel()

	var errBuf bytes.Buffer
	disp, _ := dispatchCapture(Deps{ErrOut: &errBuf})

	_ = disp.Dispatch("session-start", newSessionPayload(t))

	if !strings.Contains(errBuf.String(), "reinstall") {
		t.Errorf("expected a stderr reinstall warning with no logger, got:\n%s", errBuf.String())
	}
}

type governanceCase struct {
	name         string
	consumed     time.Duration
	delay        time.Duration
	neverReturns bool
	wantBlock    string
	wantCount    int
	wantElapsed  time.Duration
	wantSkipped  bool
}

func runGovernanceWithin(
	t *testing.T,
	testCase governanceCase,
) (governanceResult, time.Duration, bool) {
	t.Helper()

	var (
		result  governanceResult
		elapsed time.Duration
		started atomic.Bool
	)
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		defer close(release)
		disp := NewDispatcher(Deps{
			Governance: func(vcs.Repository, vcs.Status) (string, int) {
				started.Store(true)
				if !testCase.neverReturns {
					select {
					case <-time.After(testCase.delay):
						return "\nfindings", 2
					case <-release:
					}
				}
				<-release
				return "\nfindings", 2
			},
		})
		begin := time.Now()
		result = disp.governanceWithin(nil, vcs.Status{}, begin.Add(-testCase.consumed))
		elapsed = time.Since(begin)
	})
	return result, elapsed, started.Load()
}

func governanceCases() []governanceCase {
	notice := governanceTimeoutNotice()
	return []governanceCase{
		{
			name: "a run that never returns", neverReturns: true,
			wantBlock: notice, wantElapsed: stopGovernanceDeadline,
		},
		{
			name:      "a run one second past the deadline",
			delay:     stopGovernanceDeadline + time.Second,
			wantBlock: notice, wantElapsed: stopGovernanceDeadline,
		},
		{
			name: "findings just before the deadline", delay: stopGovernanceDeadline - time.Second,
			wantBlock: "\nfindings", wantCount: 2,
			wantElapsed: stopGovernanceDeadline - time.Second,
		},
		{name: "a fast run", wantBlock: "\nfindings", wantCount: 2},
		{
			name: "slow setup shortens the budget", consumed: 40 * time.Second, neverReturns: true,
			wantBlock: notice, wantElapsed: stopGovernanceDeadline - 40*time.Second,
		},
		{
			name: "a run outlasting the shortened budget", consumed: 40 * time.Second,
			delay: 25 * time.Second, wantBlock: notice,
			wantElapsed: stopGovernanceDeadline - 40*time.Second,
		},
		{
			name: "a budget spent exactly", consumed: stopGovernanceDeadline,
			neverReturns: true, wantBlock: notice, wantSkipped: true,
		},
		{
			name: "a budget overspent", consumed: stopGovernanceDeadline + time.Minute,
			delay: time.Second, wantBlock: notice, wantSkipped: true,
		},
		{
			name: "a run inside the shortened budget", consumed: 40 * time.Second,
			delay: 15 * time.Second, wantBlock: "\nfindings", wantCount: 2,
			wantElapsed: 15 * time.Second,
		},
	}
}

func TestDispatcher_GovernanceWithinDeadline(t *testing.T) {
	t.Parallel()

	notice := governanceTimeoutNotice()
	testCases := governanceCases()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			result, elapsed, started := runGovernanceWithin(t, testCase)
			want := governanceResult{
				block: testCase.wantBlock, count: testCase.wantCount,
				timedOut: testCase.wantBlock == notice,
			}
			if result != want {
				t.Errorf("got %+v, want %+v", result, want)
			}
			if elapsed != testCase.wantElapsed {
				t.Errorf("returned after %s, want %s", elapsed, testCase.wantElapsed)
			}
			if started == testCase.wantSkipped {
				t.Errorf("governance started = %v, want %v", started, !testCase.wantSkipped)
			}
		})
	}
}

func TestGovernanceTimeoutNotice_Deadline(t *testing.T) {
	t.Parallel()

	notice := governanceTimeoutNotice()
	if !strings.HasPrefix(notice, "Governance did not finish within 60s;") {
		t.Errorf("notice must state the deadline in seconds, got %q", notice)
	}
	for _, banned := range []string{"Governance checks:", "findings"} {
		if strings.Contains(notice, banned) {
			t.Errorf("a timeout must not claim findings, notice has %q: %q", banned, notice)
		}
	}
	if len(notice) > 4096 {
		t.Errorf("notice is %d bytes, over the 4096 cap", len(notice))
	}
}

type stopRun struct {
	context string
	logged  string
}

func dispatchStopWithDirtyFile(t *testing.T, delay time.Duration) stopRun {
	t.Helper()

	dir := t.TempDir()
	gitInit(t, dir)
	slowFile := filepath.Join(dir, "slow.go")
	if err := os.WriteFile(slowFile, []byte("package x\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	payload := newSessionPayload(t)
	var (
		run    stopRun
		logged bytes.Buffer
	)
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		defer close(release)
		disp, out := dispatchCapture(Deps{
			ProjectDir: dir,
			Logger:     slog.New(slog.NewTextHandler(&logged, nil)),
			Governance: func(vcs.Repository, vcs.Status) (string, int) {
				select {
				case <-time.After(delay):
				case <-release:
				}
				return "\nfinding\n", 1
			},
		})
		if err := disp.Dispatch("stop", payload); err != nil {
			t.Errorf("Dispatch must never fail on a deadline, got %v", err)
		}
		run.context = extractContext(t, out.String())
	})
	run.logged = logged.String()
	return run
}

func TestDispatchStop_GovernanceDeadline(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		delay      time.Duration
		wantIn     []string
		wantAbsent []string
		wantLogged bool
	}{
		{
			name:       "a run past the deadline",
			delay:      stopGovernanceDeadline + time.Minute,
			wantIn:     []string{"Governance did not finish within 60s", "slow.go"},
			wantAbsent: []string{"Governance findings", "Governance checks:"},
			wantLogged: true,
		},
		{
			name:       "a run that finishes",
			wantIn:     []string{"Governance findings", "slow.go"},
			wantAbsent: []string{"did not finish"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			run := dispatchStopWithDirtyFile(t, testCase.delay)
			for _, want := range testCase.wantIn {
				if !strings.Contains(run.context, want) {
					t.Errorf("stop output missing %q:\n%s", want, run.context)
				}
			}
			for _, banned := range testCase.wantAbsent {
				if strings.Contains(run.context, banned) {
					t.Errorf("stop output must not contain %q:\n%s", banned, run.context)
				}
			}
			if got := strings.Contains(run.logged, "timed_out=true"); got != testCase.wantLogged {
				t.Errorf("timed_out logged = %v, want %v:\n%s",
					got, testCase.wantLogged, run.logged)
			}
		})
	}
}

type slowStatusRepo struct {
	vcs.Repository
	delay time.Duration
}

func (repo slowStatusRepo) WorktreeStatus() (vcs.Status, error) {
	<-time.After(repo.delay)
	return vcs.Status{Changes: []vcs.Change{{Path: "slow.go", Extension: ".go"}}}, nil
}

func dispatchStopWithSlowStatus(
	t *testing.T,
	openDelay, statusDelay, governanceDelay time.Duration,
) (string, time.Duration) {
	t.Helper()

	var (
		context string
		elapsed time.Duration
	)
	payload := newSessionPayload(t)
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		defer close(release)
		disp, out := dispatchCapture(Deps{
			ProjectDir: t.TempDir(),
			Governance: func(vcs.Repository, vcs.Status) (string, int) {
				select {
				case <-time.After(governanceDelay):
				case <-release:
				}
				return "\nfinding\n", 1
			},
		})
		disp.open = func(string) (vcs.Repository, error) {
			<-time.After(openDelay)
			return slowStatusRepo{delay: statusDelay}, nil
		}
		begin := time.Now()
		if err := disp.Dispatch("stop", payload); err != nil {
			t.Errorf("Dispatch must never fail on a deadline, got %v", err)
		}
		elapsed = time.Since(begin)
		context = extractContext(t, out.String())
	})
	return context, elapsed
}

func TestDispatchStop_DeadlineFromEntry(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		openDelay       time.Duration
		statusDelay     time.Duration
		governanceDelay time.Duration
		wantNotice      bool
		wantElapsed     time.Duration
	}{
		{
			name: "slow status shortens the budget", statusDelay: 40 * time.Second,
			governanceDelay: time.Hour, wantNotice: true, wantElapsed: stopGovernanceDeadline,
		},
		{
			name: "slow open and status share the budget", openDelay: 5 * time.Second,
			statusDelay: 40 * time.Second, governanceDelay: time.Hour, wantNotice: true,
			wantElapsed: stopGovernanceDeadline,
		},
		{
			name: "governance inside the shortened budget", statusDelay: 40 * time.Second,
			governanceDelay: 15 * time.Second, wantElapsed: 55 * time.Second,
		},
		{
			name:            "status alone spends the budget",
			statusDelay:     stopGovernanceDeadline + 10*time.Second,
			governanceDelay: time.Second, wantNotice: true,
			wantElapsed: stopGovernanceDeadline + 10*time.Second,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			context, elapsed := dispatchStopWithSlowStatus(
				t, testCase.openDelay, testCase.statusDelay, testCase.governanceDelay)
			if got := strings.Contains(context, "did not finish"); got != testCase.wantNotice {
				t.Errorf("notice emitted = %v, want %v:\n%s", got, testCase.wantNotice, context)
			}
			if elapsed != testCase.wantElapsed {
				t.Errorf("Dispatch returned after %s, want %s", elapsed, testCase.wantElapsed)
			}
		})
	}
}

func TestStopGovernanceDeadline_DartBuild(t *testing.T) {
	t.Parallel()

	if dartsyntax.BuildTimeout >= stopGovernanceDeadline {
		t.Fatalf("bound %s must stay below %s", dartsyntax.BuildTimeout, stopGovernanceDeadline)
	}
}

func TestNewDispatcher_NoGovernanceStaysSilent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gitInit(t, dir)
	if err := os.WriteFile(
		filepath.Join(dir, "dirty.go"),
		[]byte("package x\n"),
		0o600,
	); err != nil {
		t.Fatalf("write: %v", err)
	}
	disp, out := dispatchCapture(Deps{ProjectDir: dir})

	_ = disp.Dispatch("stop", newSessionPayload(t))
	if out.Len() != 0 {
		t.Errorf("stop with no governance port wrote %q, want nothing", out.String())
	}
}
