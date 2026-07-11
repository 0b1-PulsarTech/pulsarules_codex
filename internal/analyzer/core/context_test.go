package core

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/astcache"
)

func newChangedGoASTsCache(t *testing.T) *astcache.Cache {
	t.Helper()

	cache := astcache.New()
	if _, err := cache.Parse("main.go", []byte("package main\n")); err != nil {
		t.Fatalf("parse main.go: %v", err)
	}
	if _, err := cache.Parse("helper_test.go", []byte("package main\n")); err != nil {
		t.Fatalf("parse helper_test.go: %v", err)
	}
	return cache
}

func TestAnalysisContext_ChangedGoASTs_NoCache(t *testing.T) {
	t.Parallel()

	ctx := &AnalysisContext{
		ChangedFiles: []FileChange{{Path: "main.go", Extension: ".go"}},
	}

	for range ctx.ChangedGoASTs() {
		t.Fatal("expected no yields when ASTCache is nil")
	}
}

func TestAnalysisContext_ChangedGoASTs_Skips(t *testing.T) {
	t.Parallel()

	cache := newChangedGoASTsCache(t)

	testCases := []struct {
		name string
		fc   FileChange
	}{
		{
			name: "non-go file skipped",
			fc:   FileChange{Path: "README.md", Extension: ".md"},
		},
		{
			name: "test file skipped",
			fc:   FileChange{Path: "helper_test.go", Extension: ".go", IsTest: true},
		},
		{
			name: "path absent from cache skipped",
			fc:   FileChange{Path: "missing.go", Extension: ".go"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := &AnalysisContext{
				ASTCache:     cache,
				ChangedFiles: []FileChange{testCase.fc},
			}
			for range ctx.ChangedGoASTs() {
				t.Fatalf("expected %s to be skipped", testCase.fc.Path)
			}
		})
	}
}

func TestAnalysisContext_ChangedGoASTs_YieldsTheRest(t *testing.T) {
	t.Parallel()

	cache := newChangedGoASTsCache(t)
	ctx := &AnalysisContext{
		ASTCache: cache,
		ChangedFiles: []FileChange{
			{Path: "README.md", Extension: ".md"},
			{Path: "helper_test.go", Extension: ".go", IsTest: true},
			{Path: "missing.go", Extension: ".go"},
			{Path: "main.go", Extension: ".go"},
		},
	}

	var got []FileChange
	for fc, f := range ctx.ChangedGoASTs() {
		if f == nil {
			t.Fatalf("expected a non-nil *ast.File for %s", fc.Path)
		}
		got = append(got, fc)
	}
	if len(got) != 1 || got[0].Path != "main.go" {
		t.Fatalf("got %v, want only main.go", got)
	}
}

func TestAnalysisContext_ChangedGoASTs_CallerBreakStops(t *testing.T) {
	t.Parallel()

	cache := newChangedGoASTsCache(t)
	if _, err := cache.Parse("second.go", []byte("package main\n")); err != nil {
		t.Fatalf("parse second.go: %v", err)
	}
	ctx := &AnalysisContext{
		ASTCache: cache,
		ChangedFiles: []FileChange{
			{Path: "main.go", Extension: ".go"},
			{Path: "second.go", Extension: ".go"},
		},
	}

	count := 0
	for range ctx.ChangedGoASTs() {
		count++
		break
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
}
