// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package handshake

import (
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
)

// LegacyHello is the legacy version of the hello command.
var LegacyHello = "isMaster"

// LegacyHelloLowercase is the lowercase, legacy version of the hello command.
var LegacyHelloLowercase = "ismaster"

func ParseClientMetadata(msg []byte) ([]byte, error) {
	command := bsoncore.Document(msg)

	// Lookup the "client" field in the command document.
	clientMetadataRaw, err := command.LookupErr("client")
	if err != nil {
		return nil, err
	}

	clientMetadata, ok := clientMetadataRaw.DocumentOK()
	if !ok {
		return nil, err
	}

	return clientMetadata, nil
}
