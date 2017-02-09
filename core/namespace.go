package core

import (
	"strings"
)

func NewNamespace(fullName string) *Namespace {
	indexOfFirstDot := strings.Index(fullName, ".")
	return &Namespace{
		DatabaseName:   fullName[:indexOfFirstDot],
		CollectionName: fullName[indexOfFirstDot + 1:],
		FullName:       fullName,
	}
}

func NewNamespaceFromDatabaseAndCollection(databaseName string, collectionName string) *Namespace {
	return &Namespace{
		DatabaseName:   databaseName,
		CollectionName: collectionName,
		FullName:       strings.Join([]string{databaseName, collectionName}, "."),
	}
}

type Namespace struct {
	DatabaseName   string
	CollectionName string
	FullName       string
}
