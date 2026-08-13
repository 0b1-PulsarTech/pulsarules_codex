package slicesx

func Dedupe[Slice ~[]Elem, Elem comparable](items Slice) Slice {
	if len(items) == 0 {
		return nil
	}
	// Order is meaningful, so slices.Compact (adjacent only, needs a sort) will not do.
	seen := make(map[Elem]struct{}, len(items))
	out := make(Slice, 0, len(items))
	for _, item := range items {
		if _, found := seen[item]; found {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}
