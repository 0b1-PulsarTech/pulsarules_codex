package cliopts

import "strings"

type stringSliceFlag []string

func (values *stringSliceFlag) String() string { return strings.Join(*values, ",") }

func (values *stringSliceFlag) Set(value string) error {
	*values = append(*values, value)
	return nil
}
