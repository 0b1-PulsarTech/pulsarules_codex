package catalog

type Catalog[Item any] struct {
	byID map[string]Item
}

func (store *Catalog[Item]) Put(id string, item Item) {
	if store.byID == nil {
		store.byID = map[string]Item{}
	}
	store.byID[id] = item
}

func (store *Catalog[Item]) Get(id string) (Item, bool) {
	item, ok := store.byID[id]
	return item, ok
}

func (store *Catalog[Item]) Has(id string) bool {
	_, ok := store.byID[id]
	return ok
}
