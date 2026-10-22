package command

import "errors"

// The command printed its findings before returning this; main exits 1 without reprinting.
var ErrFindings = errors.New("findings reported")
