package yamgo

import "fmt"

type Collection struct {
	db *Database
	name     string
}

func (coll *Collection) namespace() string {
	return fmt.Sprintf("%s.%s", coll.db.name, coll.name)
}
