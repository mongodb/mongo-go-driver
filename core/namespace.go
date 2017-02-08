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
