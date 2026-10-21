package jsondoc

import (
	"fmt"
	"os"
)

func (doc *Document) Save() error {
	return Save(doc.path, doc.fields)
}

func (doc *Document) SaveOrRemove() (removed bool, err error) {
	if doc.Len() == 0 {
		if err = os.Remove(doc.path); err != nil {
			return false, fmt.Errorf("remove %q: %w", doc.path, err)
		}
		return true, nil
	}
	if err = doc.Save(); err != nil {
		return false, fmt.Errorf("save %q: %w", doc.path, err)
	}
	return false, nil
}
