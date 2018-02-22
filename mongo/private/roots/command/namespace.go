package command

// Namespace encapsulates a database and collection name, which together
// uniquely identifies a collection within a MongoDB cluster.
type Namespace struct {
	DB         string
	Collection string
}

// NewNamespace returns a new Namespace for the
// given database and collection.
func NewNamespace(db, collection string) Namespace { return Namespace{} }

// ParseNamespace parses a namespace string into a Namespace.
//
// The namespace string must contain at least one ".", the first of which is the separator
// between the database and collection names.  If not, the default (invalid) Namespace is returned.
func ParseNamespace(name string) Namespace { return Namespace{} }

// FullName returns the full namespace string, which is the result of joining the database
// name and the collection name with a "." character.
func (*Namespace) FullName() string { return "" }
