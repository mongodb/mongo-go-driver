package core

import (
	"strings"
)

// TODO: add error checking for missing '.', etc
func ParseNamespace(fullName string) (Namespace, error) {
	indexOfFirstDot := strings.Index(fullName, ".")
	return Namespace{
		databaseName:   fullName[:indexOfFirstDot],
		collectionName: fullName[indexOfFirstDot + 1:],
	}, nil
}

// TODO: add error checking for '.' in databaseName, etc
func NewNamespace(databaseName string, collectionName string) (Namespace, error) {
	return Namespace{
		databaseName:   databaseName,
		collectionName: collectionName,
	}, nil
}

type Namespace struct {
	databaseName   string
	collectionName string
}

func (ns *Namespace) DatabaseName() string {
	return ns.databaseName
}

func (ns *Namespace) CollectionName() string {
	return ns.collectionName
}

func (ns *Namespace) FullName() string {
	return strings.Join([]string{ns.databaseName, ns.collectionName}, ".")
}
