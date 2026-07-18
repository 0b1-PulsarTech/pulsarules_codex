package safeyaml

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

func Unmarshal(raw []byte, out any) error {
	return guard(func() error { return yaml.Unmarshal(raw, out) })
}

func DecodeNode(node *yaml.Node, out any) error {
	return guard(func() error { return node.Decode(out) })
}

func DecodeNext(decoder *yaml.Decoder, out any) error {
	return guard(func() error { return decoder.Decode(out) })
}

func guard(call func() error) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			// yaml.v3 panics on some well-formed documents; the "yaml: " prefix lets callers
			// scrub the text like any other decoder error.
			err = fmt.Errorf("yaml: %v", recovered)
		}
	}()
	return call()
}
