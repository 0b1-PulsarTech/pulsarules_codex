package jsondoc

import (
	"errors"
	"fmt"
)

type parseError struct{ cause error }

func (failure *parseError) Error() string { return failure.cause.Error() }

func (failure *parseError) Unwrap() error { return failure.cause }

func (failure *parseError) Is(target error) bool { return target == ErrUnparseable }

func PrefixParse(err error, prefix string) error {
	// Only a parse failure takes the caller's wording; a read failure stays as it came.
	if !errors.Is(err, ErrUnparseable) {
		return err
	}
	return fmt.Errorf("%s: %w", prefix, err)
}
