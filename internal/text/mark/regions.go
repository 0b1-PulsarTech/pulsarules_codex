package mark

import "slices"

func unionAll(lists [][]Region) []Region {
	var all []Region
	for _, list := range lists {
		all = append(all, list...)
	}
	slices.SortFunc(all, func(left, right Region) int { return left.Start - right.Start })
	merged := all[:0]
	for _, region := range all {
		last := len(merged) - 1
		if last >= 0 && region.Start <= merged[last].End {
			merged[last].End = max(merged[last].End, region.End)
			continue
		}
		merged = append(merged, region)
	}
	return merged
}

func intersectAll(lists [][]Region) []Region {
	if len(lists) == 0 {
		return nil
	}
	shared := lists[0]
	for _, list := range lists[1:] {
		shared = intersect(shared, list)
	}
	return shared
}

func intersect(left, right []Region) []Region {
	var both []Region
	for lidx, ridx := 0, 0; lidx < len(left) && ridx < len(right); {
		start := max(left[lidx].Start, right[ridx].Start)
		end := min(left[lidx].End, right[ridx].End)
		if start < end {
			both = append(both, Region{Start: start, End: end})
		}
		if left[lidx].End < right[ridx].End {
			lidx++
		} else {
			ridx++
		}
	}
	return both
}
