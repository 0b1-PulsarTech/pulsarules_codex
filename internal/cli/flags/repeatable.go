package flags

import (
	"slices"
	"strings"
)

type Repeatable struct {
	target *[]string
}

func NewRepeatable(target *[]string) *Repeatable { return &Repeatable{target: target} }

func (values *Repeatable) String() string {
	if values == nil || values.target == nil {
		return ""
	}
	return strings.Join(*values.target, ",")
}

func (values *Repeatable) Set(raw string) error {
	if !slices.Contains(*values.target, raw) {
		// Kept exactly as typed, so validation sees "a,b" and " a " as given; only an exact
		// repeat is dropped.
		*values.target = append(*values.target, raw)
	}
	return nil
}
