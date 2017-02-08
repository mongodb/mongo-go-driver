package core

import (
	"strings"
)

func NewNamespace(fullName string) *Namespace {
	indexOfFirstDot := strings.Index(fullName, ".")
	return &Namespace{
		databaseName:   fullName[:indexOfFirstDot],
		collectionName: fullName[indexOfFirstDot + 1:],
		fullName:       fullName,
	}
}

type Namespace struct {
	databaseName   string
	collectionName string
	fullName       string
}

func (ns *Namespace) DatabaseName() string {
	return ns.databaseName
}

func (ns *Namespace) CollectionName() string {
	return ns.collectionName
}

func (ns *Namespace) FullName() string {
	return ns.fullName
}
