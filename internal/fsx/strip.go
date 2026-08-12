package fsx

import (
	"encoding/json"
	"errors"
	"fmt"
)

var ErrUnparseableJSON = errors.New("existing config file is not valid JSON")

func StripMapSection[Mapping ~map[Key]Val, Key comparable, Val any](
	config map[string]json.RawMessage, key, desc string, strip func(Mapping) bool,
) (bool, error) {
	return stripSection(config, section[Mapping]{
		key:   key,
		desc:  desc,
		strip: func(value *Mapping) bool { return strip(*value) },
		empty: func(value Mapping) bool { return len(value) == 0 },
	})
}

func StripSliceSection[Slice ~[]Elem, Elem any](
	config map[string]json.RawMessage, key, desc string, strip func(*Slice) bool,
) (bool, error) {
	// The callback takes *Slice to store back what slices.DeleteFunc returns.
	return stripSection(config, section[Slice]{
		key:   key,
		desc:  desc,
		strip: strip,
		empty: func(value Slice) bool { return len(value) == 0 },
	})
}

type section[Value any] struct {
	key, desc string
	strip     func(*Value) bool
	empty     func(Value) bool
}

func stripSection[Value any](
	config map[string]json.RawMessage, spec section[Value],
) (bool, error) {
	key := spec.key
	raw, ok := config[key]
	if !ok {
		return false, nil
	}
	var value Value
	if err := json.Unmarshal(raw, &value); err != nil {
		// Callers warn and leave the file untouched rather than rewrite content they cannot read.
		return false, fmt.Errorf("%w: %s: %w", ErrUnparseableJSON, spec.desc, err)
	}
	if !spec.strip(&value) {
		return false, nil
	}
	if spec.empty(value) {
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
