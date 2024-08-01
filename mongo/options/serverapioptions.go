// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

import (
	"fmt"
)

// ServerAPIOptions represents arguments used to configure the API version sent to
// the server when running commands.
//
// Sending a specified server API version causes the server to behave in a
// manner compatible with that API version. It also causes the driver to behave
// in a manner compatible with the driver’s behavior as of the release when the
// driver first started to support the specified server API version.
//
// The user must specify a ServerAPIVersion if including ServerAPIOptions in
// their client. That version must also be currently supported by the driver.
// This version of the driver supports API version "1".
type ServerAPIOptions struct {
	ServerAPIVersion  ServerAPIVersion
	Strict            *bool
	DeprecationErrors *bool
}

// ServerAPIOptionsBuilder contains options to configure serverAPI operations.
// Each option can be set through setter functions. See documentation for each
// setter function for an explanation of the option.
type ServerAPIOptionsBuilder struct {
	Opts []func(*ServerAPIOptions) error
}

// ServerAPI creates a new ServerAPIOptions configured with the provided
// serverAPIversion.
func ServerAPI(serverAPIVersion ServerAPIVersion) *ServerAPIOptionsBuilder {
	opts := &ServerAPIOptionsBuilder{}

	opts.Opts = append(opts.Opts, func(opts *ServerAPIOptions) error {
		opts.ServerAPIVersion = serverAPIVersion

		return nil
	})

	return opts
}

// List returns a list of ServerAPIOptions setter functions.
func (s *ServerAPIOptionsBuilder) List() []func(*ServerAPIOptions) error {
	return s.Opts
}

// SetStrict specifies whether the server should return errors for features that are not part of the API version.
func (s *ServerAPIOptionsBuilder) SetStrict(strict bool) *ServerAPIOptionsBuilder {
	s.Opts = append(s.Opts, func(opts *ServerAPIOptions) error {
		opts.Strict = &strict

		return nil
	})

	return s
}

// SetDeprecationErrors specifies whether the server should return errors for deprecated features.
func (s *ServerAPIOptionsBuilder) SetDeprecationErrors(deprecationErrors bool) *ServerAPIOptionsBuilder {
	s.Opts = append(s.Opts, func(opts *ServerAPIOptions) error {
		opts.DeprecationErrors = &deprecationErrors

		return nil
	})

	return s
}

// ServerAPIVersion represents an API version that can be used in ServerAPIOptions.
type ServerAPIVersion string

const (
	// ServerAPIVersion1 is the first API version.
	ServerAPIVersion1 ServerAPIVersion = "1"
)

// Validate determines if the provided ServerAPIVersion is currently supported by the driver.
func (sav ServerAPIVersion) Validate() error {
	if sav == ServerAPIVersion1 {
		return nil
	}
	return fmt.Errorf("api version %q not supported; this driver version only supports API version \"1\"", sav)
}
