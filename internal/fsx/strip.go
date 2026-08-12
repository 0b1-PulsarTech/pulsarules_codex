package fsx

import (
	"encoding/json"
	"errors"
	"fmt"
)

// ErrUnparseableJSON marks a config file a wire package could not parse as
// JSON. The caller must warn and leave the file untouched rather than risk
// rewriting hand-edited content it cannot safely read back.
var ErrUnparseableJSON = errors.New("existing config file is not valid JSON")

// StripMapSection decodes config's key section into M, runs strip over it in
// place (a map is a reference type, no reassignment needed), then deletes
// key once empty or re-marshals the trimmed map back into config. desc names
// the section for the ErrUnparseableJSON message. No-op when key is absent.
func StripMapSection[M ~map[K]V, K comparable, V any](
	config map[string]json.RawMessage, key, desc string, strip func(M) bool,
) (bool, error) {
	return stripSection(config, key, desc,
		func(value *M) bool { return strip(*value) },
		func(value M) bool { return len(value) == 0 },
	)
}

// StripSliceSection decodes config's key section into S, runs strip over it
// (which may reassign the slice, e.g. via slices.DeleteFunc, hence the
// pointer), then deletes key once empty or re-marshals the trimmed slice
// back into config. desc names the section for the ErrUnparseableJSON
// message. No-op when key is absent.
func StripSliceSection[S ~[]E, E any](
	config map[string]json.RawMessage, key, desc string, strip func(*S) bool,
) (bool, error) {
	return stripSection(config, key, desc, strip, func(value S) bool { return len(value) == 0 })
}

// why: shared decode/strip/marshal-or-delete core; StripMapSection and
// StripSliceSection each supply their own pointer shape and emptiness check.
func stripSection[T any](
	config map[string]json.RawMessage, key, desc string,
	strip func(*T) bool, empty func(T) bool,
) (bool, error) {
	raw, ok := config[key]
	if !ok {
		return false, nil
	}
	var value T
	if err := json.Unmarshal(raw, &value); err != nil {
		return false, fmt.Errorf("%w: %s: %w", ErrUnparseableJSON, desc, err)
	}
	if !strip(&value) {
		return false, nil
	}
	if empty(value) {
		delete(config, key)
		return true, nil
	}
	marshaled, err := json.Marshal(value)
	if err != nil {
		return false, fmt.Errorf("marshal %s: %w", key, err)
	}
	config[key] = marshaled
	return true, nil
}
