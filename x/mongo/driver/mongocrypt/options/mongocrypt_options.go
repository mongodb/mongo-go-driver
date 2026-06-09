// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

import (
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
)

// MongoCryptOptions specifies options to configure a MongoCrypt instance.
type MongoCryptOptions struct {
	KmsProviders               bsoncore.Document
	LocalSchemaMap             map[string]bsoncore.Document
	BypassQueryAnalysis        bool
	EncryptedFieldsMap         map[string]bsoncore.Document
	CryptSharedLibDisabled     bool
	CryptSharedLibOverridePath string
	HTTPClient                 *http.Client
	KeyExpiration              *time.Duration
}
