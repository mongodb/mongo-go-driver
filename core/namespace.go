package core

import (
	"errors"
	"strings"
)

// ParseNamesepace parses a namespace string into a Namespace.
//
// The namespace string must contain at least one ".", the first of which is the separator
// between the database and collection names.  After the namespace string is split,
// the rules in NewNamespace are applied.
func ParseNamespace(fullName string) (*Namespace, error) {
	indexOfFirstDot := strings.Index(fullName, ".")
	if indexOfFirstDot == -1 {
		return nil, errors.New("Namespace must contain a '.'")
	}
	databaseName := fullName[:indexOfFirstDot]
	collectionName := fullName[indexOfFirstDot + 1:]
	return NewNamespace(databaseName, collectionName)
}

// NewNamespace creates a Namespace from the given database and collection names.
//
// Neither can be empty, and the database name may not contain a "." or " " character
func NewNamespace(databaseName string, collectionName string) (*Namespace, error) {
	if collectionName == "" {
		return nil, errors.New("Collection name can not be empty")
	}
	if databaseName == "" {
		return nil, errors.New("Database name can not be empty")
	}
	if strings.Contains(databaseName, " ") {
		return nil, errors.New("Database name can not contain ' '")
	}
	if strings.Contains(databaseName, ".") {
		return nil, errors.New("Database name can not contain '.'")
	}

	return &Namespace{
		databaseName:   databaseName,
		collectionName: collectionName,
	}, nil
}

type Namespace struct {
	databaseName   string
	collectionName string
}

// DatabaseName returns the name of the database.
func (ns *Namespace) DatabaseName() string {
	return ns.databaseName
}

// CollectionName returns the name of the collection.
func (ns *Namespace) CollectionName() string {
	return ns.collectionName
}

// FullName returns the full namespace string, which is the result of joining the database
// name and the collection name with a "." character.
func (ns *Namespace) FullName() string {
	return strings.Join([]string{ns.databaseName, ns.collectionName}, ".")
}
