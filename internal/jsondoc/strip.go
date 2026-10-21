package jsondoc

func StripMap[Mapping ~map[Key]Val, Key comparable, Val any](
	doc *Document, key string, strip func(Mapping) bool,
) (bool, error) {
	return stripSection(doc, section[Mapping]{
		key:   key,
		strip: func(value *Mapping) bool { return strip(*value) },
		empty: func(value Mapping) bool { return len(value) == 0 },
	})
}

func StripSlice[Slice ~[]Elem, Elem any](
	doc *Document, key string, strip func(*Slice) bool,
) (bool, error) {
	// The callback takes *Slice to store back what slices.DeleteFunc returns.
	return stripSection(doc, section[Slice]{
		key:   key,
		strip: strip,
		empty: func(value Slice) bool { return len(value) == 0 },
	})
}

type section[Value any] struct {
	key   string
	strip func(*Value) bool
	empty func(Value) bool
}

func stripSection[Value any](doc *Document, spec section[Value]) (bool, error) {
	if !doc.Has(spec.key) {
		return false, nil
	}
	// Callers warn and leave the file untouched rather than rewrite content they cannot read.
	value, err := Section[Value](doc, spec.key)
	if err != nil {
		return false, err
	}
	if !spec.strip(&value) {
		return false, nil
	}
	if spec.empty(value) {
		doc.Delete(spec.key)
		return true, nil
	}
	if err = SetSection(doc, spec.key, value); err != nil {
		return false, err
	}
	return true, nil
}
