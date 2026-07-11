package arch

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// projectIndex holds the parsed package map and the built dependency graph
// for one project. Both arch analyzers share this instead of each calling
// discoverPackages + buildGraph independently.
type projectIndex struct {
	pkgMap map[string]*pkgImports
	graph  depGraph
}

// cacheEntry holds a projectIndex plus the invalidation data used to decide
// whether the entry is still fresh.
type cacheEntry struct {
	index     *projectIndex
	goModHash string
	fileCount int
	builtAt   time.Time
}

// indexCache is a process-level map from project directory to its cached
// project index. It is safe for concurrent access.
var indexCache sync.Map

// cacheTTL is how long an entry stays fresh without re-validation.
const cacheTTL = 30 * time.Second

// loadProjectIndex returns the cached project index for projectDir, or builds
// and caches a fresh one. The cache is invalidated when go.mod changes, the
// number of .go files changes, or the TTL expires.
func loadProjectIndex(projectDir, modulePath string) *projectIndex {
	goModHash := hashGoMod(projectDir)
	fileCount := countGoFiles(projectDir)

	if entry, ok := indexCache.Load(projectDir); ok {
		cached := entry.(*cacheEntry)
		if cached.goModHash == goModHash &&
			cached.fileCount == fileCount &&
			time.Since(cached.builtAt) < cacheTTL {
			return cached.index
		}
	}

	pkgMap := discoverPackages(projectDir, modulePath)
	graph := buildGraph(pkgMap, modulePath)

	idx := &projectIndex{pkgMap: pkgMap, graph: graph}
	indexCache.Store(projectDir, &cacheEntry{
		index:     idx,
		goModHash: goModHash,
		fileCount: fileCount,
		builtAt:   time.Now(),
	})
	return idx
}

// hashGoMod returns a hex hash of the go.mod file contents, or empty string
// when go.mod is absent or unreadable.
func hashGoMod(projectDir string) string {
	//nolint:gosec // path comes from trusted project config
	content, err := os.ReadFile(filepath.Join(projectDir, "go.mod"))
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

// countGoFiles walks the project directory and returns the number of non-test
// .go files, skipping vendor, hidden dirs, and node_modules.
func countGoFiles(projectDir string) int {
	count := 0
	_ = filepath.WalkDir(projectDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if strings.HasPrefix(name, ".") || name == "vendor" || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			count++
		}
		return nil
	})
	return count
}
