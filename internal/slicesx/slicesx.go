package slicesx

// Dedupe returns s with duplicates removed, preserving first-seen order.
// why: slices.Compact only collapses ADJACENT elements and needs a sort
// first - which install/hook/workflow order callers must not do, since
// order is meaningful to the user. Three call sites had hand-rolled this.
func Dedupe[S ~[]E, E comparable](s S) S {
	if len(s) == 0 {
		return nil
	}
	seen := make(map[E]struct{}, len(s))
	out := make(S, 0, len(s))
	for _, item := range s {
		if _, found := seen[item]; found {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}
