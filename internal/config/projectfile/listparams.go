package projectfile

import (
	"fmt"
	"maps"
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config/projectfile/strictyaml"
)

func (val *validator) checkListParams(field *strictyaml.Field, params map[string]any) {
	for _, key := range slices.Sorted(maps.Keys(params)) {
		items, isList := params[key].([]any)
		if !isList {
			continue
		}
		// ParamSet.Strings drops a non-string item and keeps the default list without a word.
		for idx, item := range items {
			if _, isText := item.(string); !isText {
				val.fail(field.Key(key), "%s", offendingItem(idx, item))
				break
			}
		}
	}
}

func offendingItem(idx int, item any) string {
	switch item.(type) {
	case int, int64, uint64, float64:
		return fmt.Sprintf("item %d is a number; quote it (%q)", idx+1, fmt.Sprint(item))
	case bool:
		return fmt.Sprintf("item %d is a boolean; want a string", idx+1)
	case nil:
		return fmt.Sprintf("item %d is empty; want a string", idx+1)
	}
	return fmt.Sprintf("item %d is a nested value; want a string", idx+1)
}
