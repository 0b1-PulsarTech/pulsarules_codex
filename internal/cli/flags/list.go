package flags

import "strings"

type CommaList struct {
	target *[]string
}

func NewCommaList(target *[]string) *CommaList { return &CommaList{target: target} }

func (list *CommaList) String() string {
	if list == nil || list.target == nil {
		return ""
	}
	return strings.Join(*list.target, ",")
}

func (list *CommaList) Set(raw string) error {
	// Last-wins: a repeated flag replaces the whole list, never extends it.
	*list.target = nil
	for part := range strings.SplitSeq(raw, ",") {
		if name := strings.TrimSpace(part); name != "" {
			*list.target = append(*list.target, name)
		}
	}
	// "" means not given, but a blank value such as " " is given and empty.
	if raw != "" && *list.target == nil {
		*list.target = []string{}
	}
	return nil
}
