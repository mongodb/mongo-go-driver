// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package ops

import (
	"errors"
	"strings"
)

// NewNamespace returns a new Namespace for the
// given database and collection.
func NewNamespace(db, collection string) Namespace {
	return Namespace{
		DB:         db,
		Collection: collection,
	}
}

// ParseNamespace parses a namespace string into a Namespace.
//
// The namespace string must contain at least one ".", the first of which is the separator
// between the database and collection names.  If not, the default (invalid) Namespace is returned.
func ParseNamespace(fullName string) Namespace {
	indexOfFirstDot := strings.Index(fullName, ".")
	if indexOfFirstDot == -1 {
		return Namespace{}
	}
	return Namespace{
		DB:         fullName[:indexOfFirstDot],
		Collection: fullName[indexOfFirstDot+1:],
	}
}

// Namespace encapsulates a database and collection name, which together uniquely identifies a collection within a MongoDB cluster.
type Namespace struct {
	DB         string
	Collection string
}

// FullName returns the full namespace string, which is the result of joining the database
// name and the collection name with a "." character.
func (ns *Namespace) FullName() string {
	return strings.Join([]string{ns.DB, ns.Collection}, ".")
}

// Validates the namespace.
func (ns *Namespace) validate() error {
	if err := validateDB(ns.DB); err != nil {
		return err
	}

	return validateCollection(ns.Collection)
}

// The database name can not be the empty string, and may not contain a "." or " " character.
func validateDB(DB string) error {
	if DB == "" {
		return errors.New("database name can not be empty")
	}
	if strings.Contains(DB, " ") {
		return errors.New("database name can not contain ' '")
	}
	if strings.Contains(DB, ".") {
		return errors.New("database name can not contain '.'")
	}

	return nil
}

// The collection name can not be the empty string.
func validateCollection(collection string) error {
	if collection == "" {
		return errors.New("database name can not be empty")
	}

	return nil
}
