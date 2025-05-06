// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mtest

// SetupOptions is the type used to configure mtest setup
type SetupOptions struct {
	// Specifies the URI to connect to. Defaults to URI based on the environment variables MONGODB_URI,
	// MONGO_GO_DRIVER_CA_FILE, and MONGO_GO_DRIVER_COMPRESSOR
	URI *string
}

// NewSetupOptions creates an empty SetupOptions struct
func NewSetupOptions() *SetupOptions {
	return &SetupOptions{}
}

// SetURI sets the uri to connect to
func (so *SetupOptions) SetURI(uri string) *SetupOptions {
	so.URI = &uri
	return so
}
