package emoji

import (
	"hash/fnv"
	"slices"
)

type candidateSet struct {
	themed   []string
	typed    []string
	measured []string
	fresh    []string
}

func (catalog *Catalog) Suggest(commitType, seed string, exclude []string, count int) []string {
	if count <= 0 {
		return nil
	}
	candidates := catalog.candidates(commitType, seed, exclude)

	const themedShare = 2
	// At most count/2+1 come from the change's own emoji family; the rest keeps the set varied.
	picks := drawSpread(candidates.themed, seed, min(count/themedShare+1, count))
	for _, tier := range [][]string{candidates.typed, candidates.measured, candidates.fresh} {
		if len(picks) >= count {
			break
		}
		picks = append(picks, drawSpread(tier, seed, count-len(picks))...)
	}
	return picks
}

func drawSpread(ranked []string, seed string, want int) []string {
	if want <= 0 || len(ranked) == 0 {
		return nil
	}
	if len(ranked) <= want {
		return slices.Clone(ranked)
	}

	offset := int(hashSeed(seed))
	bandSize := len(ranked) / want
	picks := make([]string, 0, want)
	// A tier contributes across its whole range, not only its head.
	for band := range want {
		picks = append(picks, ranked[band*bandSize+offset%bandSize])
	}
	return picks
}

type picker struct {
	seen    map[string]bool
	exclude []string
	inPool  map[string]bool
}

func (pick picker) take(shortcode string) bool {
	if pick.seen[shortcode] || !pick.inPool[shortcode] {
		return false
	}
	if slices.Contains(pick.exclude, shortcode) {
		return false
	}
	pick.seen[shortcode] = true
	return true
}

func (catalog *Catalog) candidates(commitType, seed string, exclude []string) candidateSet {
	var set candidateSet
	pick := picker{
		seen:    make(map[string]bool, len(catalog.pool)),
		exclude: exclude,
		inPool:  catalog.inPool,
	}

	for _, shortcode := range catalog.themeMatches(seed) {
		if pick.take(shortcode) {
			set.themed = append(set.themed, shortcode)
		}
	}
	// The type ranking only biases order: the emoji names the area of the change, not its type.
	for _, shortcode := range catalog.byType[commitType] {
		if pick.take(shortcode) {
			set.typed = append(set.typed, shortcode)
		}
	}
	for _, shortcode := range catalog.pool {
		if !pick.take(shortcode) {
			continue
		}
		if catalog.allowed[shortcode].janeCount > 0 {
			set.measured = append(set.measured, shortcode)
		} else {
			set.fresh = append(set.fresh, shortcode)
		}
	}
	return set
}

func hashSeed(seed string) uint32 {
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(seed))
	return hasher.Sum32()
}
