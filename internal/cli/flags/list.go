package flags

import "strings"

type List struct {
	target *[]string
	isSet  bool
}

func NewList(target *[]string) *List { return &List{target: target} }

func (list *List) String() string {
	if list == nil || list.target == nil {
		return ""
	}
	return strings.Join(*list.target, ",")
}

func (list *List) Set(raw string) error {
	if !list.isSet {
		// The first value replaces any default the target held.
		*list.target = nil
		list.isSet = true
	}
	for part := range strings.SplitSeq(raw, ",") {
		if name := strings.TrimSpace(part); name != "" {
			*list.target = append(*list.target, name)
		}
	}
	// "--skills ' '" and "--skills ''" were different inputs before lists were shared:
	// only the blank one counts as given, so a non-empty raw value never leaves nil.
	if raw != "" && *list.target == nil {
		*list.target = []string{}
	}
	return nil
}
