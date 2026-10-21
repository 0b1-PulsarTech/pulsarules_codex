package flags

import "strings"

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
	// Each occurrence stays exactly as typed, so validation sees "a,b" and " a " as given.
	*values.target = append(*values.target, raw)
	return nil
}
