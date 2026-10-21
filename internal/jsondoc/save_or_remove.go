package jsondoc

import (
	"fmt"
	"os"
)

func (doc *Document) Save() error {
	return Save(doc.path, doc.fields)
}

func (doc *Document) SaveOrRemove() error {
	if doc.Len() == 0 {
		if err := os.Remove(doc.path); err != nil {
			return fmt.Errorf("remove %q: %w", doc.path, err)
		}
		return nil
	}
	if err := doc.Save(); err != nil {
		return fmt.Errorf("save %q: %w", doc.path, err)
	}
	return nil
}
