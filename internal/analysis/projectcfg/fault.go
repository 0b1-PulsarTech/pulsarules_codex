package projectcfg

type reworded struct {
	text  string
	cause error
}

func (fault *reworded) Error() string { return fault.text }

func (fault *reworded) Unwrap() error { return fault.cause }
