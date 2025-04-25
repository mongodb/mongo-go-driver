// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package xoptions

import (
	"fmt"

	internalOptions "go.mongodb.org/mongo-driver/v2/internal/options"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
)

// SetInternalClientOptions sets internal options for ClientOptions.
//
// Deprecated: This function is for internal use only. It may be changed or removed in any release.
func SetInternalClientOptions(opts *options.ClientOptions, key string, option any) error {
	const typeErr = "unexpected type for %s"
	switch key {
	case "crypt":
		c, ok := option.(driver.Crypt)
		if !ok {
			return fmt.Errorf(typeErr, key)
		}
		opts.Crypt = c
	case "deployment":
		d, ok := option.(driver.Deployment)
		if !ok {
			return fmt.Errorf(typeErr, key)
		}
		opts.Deployment = d
	case "authenticateToAnything":
		b, ok := option.(bool)
		if !ok {
			return fmt.Errorf(typeErr, key)
		}
		opts.Custom = internalOptions.WithValue(opts.Custom, key, b)
	default:
		return fmt.Errorf("unsupported option: %s", key)
	}
	return nil
}
