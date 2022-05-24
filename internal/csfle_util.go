// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package internal

import (
	"fmt"

	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

type StateCollection uint8

const (
	EncryptedCacheCollection StateCollection = iota
	EncryptedStateCollection
	EncryptedCompactionCollection
)

func (sc StateCollection) suffix() string {
	switch sc {
	case EncryptedCacheCollection:
		return "ecc"
	case EncryptedStateCollection:
		return "esc"
	case EncryptedCompactionCollection:
		return "ecoc"
	}
	return "unknown"
}

// GetEncryptedStateCollectionName returns the encrypted state collection name associated with dataCollectionName.
func GetEncryptedStateCollectionName(efBSON bsoncore.Document, dataCollectionName string, sc StateCollection) (string, error) {
	fieldName := sc.suffix() + "Collection"
	val, err := efBSON.LookupErr(fieldName)
	if err != nil {
		if err != bsoncore.ErrElementNotFound {
			return "", err
		}
		// Return default name.
		defaultName := "enxcol_." + dataCollectionName + "." + sc.suffix()
		return defaultName, nil
	}

	stateCollectionName, ok := val.StringValueOK()
	if !ok {
		return "", fmt.Errorf("expected string for '%v', got: %v", fieldName, val.Type)
	}
	return stateCollectionName, nil
}
