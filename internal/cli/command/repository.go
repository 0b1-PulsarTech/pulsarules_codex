package command

import (
	"errors"
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

//nolint:ireturn // returns the consumer port, nil when there is no repository
func OptionalRepository(open func() (*vcs.Repository, error)) (governance.Repository, error) {
	opened, err := open()
	if err != nil && !errors.Is(err, vcs.ErrNoRepository) {
		return nil, fmt.Errorf("open repository: %w", err)
	}
	// The one place an absent repository becomes a true nil interface: a nil
	// *vcs.Repository stored in the port would not compare equal to nil.
	if opened == nil {
		return nil, nil //nolint:nilnil // an absent repository is a valid, explicit nil
	}
	return opened, nil
}
