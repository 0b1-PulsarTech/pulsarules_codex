package catalog

import "iter"

type Catalog[Item any] struct {
	items []Item
	byID  map[string]int
}

func (store *Catalog[Item]) Add(id string, item Item) bool {
	if _, dup := store.byID[id]; dup {
		return false
	}
	if store.byID == nil {
		store.byID = map[string]int{}
	}
	store.byID[id] = len(store.items)
	store.items = append(store.items, item)
	return true
}

func (store *Catalog[Item]) Replace(id string, item Item) {
	if pos, ok := store.byID[id]; ok {
		store.items[pos] = item
	}
}

func (store *Catalog[Item]) Get(id string) (Item, bool) {
	pos, ok := store.byID[id]
	if !ok {
		var zero Item
		return zero, false
	}
	return store.items[pos], true
}

func (store *Catalog[Item]) Has(id string) bool {
	_, ok := store.byID[id]
	return ok
}

func (store *Catalog[Item]) All() iter.Seq[Item] {
	return func(yield func(Item) bool) {
		for _, item := range store.items {
			if !yield(item) {
				return
			}
		}
	}
}
