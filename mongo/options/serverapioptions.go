// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

import "fmt"

// ServerAPIOptions represents options used to configure the API version sent to the server
// when running commands.
//
// Sending a specified server API version causes the server to behave in a manner compatible with that
// API version. It also causes the driver to behave in a manner compatible with the driver’s behavior as
// of the release when the driver first started to support the specified server API version.
//
// The user must specify a ServerAPIVersion if including ServerAPIOptions in their client. That version
// must also be currently supported by the driver.
type ServerAPIOptions struct {
	// ServerAPIVersion is the version string of the declared API version
	ServerAPIVersion ServerAPIVersion
	// Strict determines whether the server should return errors for features that are not part of the API version
	Strict bool
	// DeprecationErrors determines whether the server should return errors for deprecated features
	DeprecationErrors bool
}

// ServerAPI creates a new ServerAPIOptions configured with default values.
func ServerAPI() *ServerAPIOptions {
	return &ServerAPIOptions{ServerAPIVersion: ServerAPIVersion1}
}

// ServerAPIVersion represents an API version that can be used in ServerAPIOptions.
type ServerAPIVersion string

const (
	// ServerAPIVersion1 is the first API version.
	ServerAPIVersion1 ServerAPIVersion = "1"
)

// CheckServerAPIVersion determines if the provided ServerAPIVersion is currently supported by the driver.
func (version ServerAPIVersion) CheckServerAPIVersion() error {
	switch version {
	case ServerAPIVersion1:
		return nil
	}
	return fmt.Errorf("api version %v not supported by driver", version)
}
