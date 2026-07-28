package emoji

import (
	"hash/fnv"
	"slices"
)

// themedShare is the fraction of a suggestion set drawn from the emoji family
// matching the change itself. Relevance is the point of a suggestion; the
// remainder keeps the vocabulary from narrowing.
const themedShare = 2

// candidateSet is the suggestion pool split by how well each shortcode fits
// the change at hand: the family matching the subject, the commit type's own
// measured vocabulary, everything else with measured usage, and the unused.
type candidateSet struct {
	themed   []string
	typed    []string
	measured []string
	fresh    []string
}

// Suggest returns up to count shortcodes to offer as alternatives.
//
// The set leads with emoji whose family matches what the subject describes,
// then the commit type's measured vocabulary, then the wider pool. The seed
// varies the picks between commits while staying reproducible for the same
// message - always naming the same few alternatives is what produced the runs
// of identical emoji this catalog exists to break up. Shortcodes in exclude
// (the recent-history window) are never offered.
func (c *Catalog) Suggest(commitType, seed string, exclude []string, count int) []string {
	if count <= 0 {
		return nil
	}
	candidates := c.candidates(commitType, seed, exclude)

	picks := drawSpread(candidates.themed, seed, min(count/themedShare+1, count))
	for _, tier := range [][]string{candidates.typed, candidates.measured, candidates.fresh} {
		if len(picks) >= count {
			break
		}
		picks = append(picks, drawSpread(tier, seed, count-len(picks))...)
	}
	return picks
}

// drawSpread slices ranked into want contiguous bands and takes one entry from
// each, so a tier contributes across its whole range rather than only its head.
// The seed rotates the pick inside every band.
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
	for band := range want {
		picks = append(picks, ranked[band*bandSize+offset%bandSize])
	}
	return picks
}

// candidates splits the pool into relevance tiers. The type ranking only
// biases the order - the emoji names the AREA of the change, not its type, so
// no type is ever confined to its own bucket.
func (c *Catalog) candidates(commitType, seed string, exclude []string) candidateSet {
	var set candidateSet
	seen := make(map[string]bool, len(c.pool))
	take := func(shortcode string) bool {
		if seen[shortcode] || slices.Contains(exclude, shortcode) || !c.inPool[shortcode] {
			return false
		}
		seen[shortcode] = true
		return true
	}

	for _, shortcode := range themeMatches(seed) {
		if take(shortcode) {
			set.themed = append(set.themed, shortcode)
		}
	}
	for _, shortcode := range c.byType[commitType] {
		if take(shortcode) {
			set.typed = append(set.typed, shortcode)
		}
	}
	for _, shortcode := range c.pool {
		if !take(shortcode) {
			continue
		}
		if c.allowed[shortcode].janeCount > 0 {
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
