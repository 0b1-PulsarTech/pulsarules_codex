package vcs

type Side int

const (
	Head Side = iota
	Staged
)

func (side Side) String() string {
	if side == Staged {
		return "staged"
	}
	return "HEAD"
}
