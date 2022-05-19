// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package internal // import "go.mongodb.org/mongo-driver/internal"

import (
	"fmt"

	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

// GetEncryptedStateCollectionName returns the encrypted state collection name associated with dataCollectionName.
func GetEncryptedStateCollectionName(efBSON bsoncore.Document, dataCollectionName string, stateCollectionSuffix string) (string, error) {
	if stateCollectionSuffix != "esc" && stateCollectionSuffix != "ecc" && stateCollectionSuffix != "ecoc" {
		return "", fmt.Errorf("expected stateCollectionSuffix: esc, ecc, or ecoc. got %v", stateCollectionSuffix)
	}
	fieldName := stateCollectionSuffix + "Collection"
	var val bsoncore.Value
	var err error
	if val, err = efBSON.LookupErr(fieldName); err != nil {
		if err != bsoncore.ErrElementNotFound {
			return "", err
		}
		// Return default name.
		defaultName := "enxcol_." + dataCollectionName + "." + stateCollectionSuffix
		return defaultName, nil
	}

	var stateCollectionName string
	var ok bool
	if stateCollectionName, ok = val.StringValueOK(); !ok {
		return "", fmt.Errorf("expected string for '%v', got: %v", fieldName, val.Type)
	}
	return stateCollectionName, nil
}
