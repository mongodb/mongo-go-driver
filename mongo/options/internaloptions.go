// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

import (
	"fmt"

	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
)

// Deprecated: SetInternalClientOptions sets internal only options for ClientOptions. It may be changed
// or removed in any release.
func SetInternalClientOptions(opts *ClientOptions, custom map[string]any) (*ClientOptions, error) {
	const typeErr = "unexpected type for %s"
	for k, v := range custom {
		switch k {
		case "crypt":
			c, ok := v.(driver.Crypt)
			if !ok {
				return nil, fmt.Errorf(typeErr, k)
			}
			opts.Crypt = c
		case "deployment":
			d, ok := v.(driver.Deployment)
			if !ok {
				return nil, fmt.Errorf(typeErr, k)
			}
			opts.Deployment = d
		default:
			return nil, fmt.Errorf("unsupported option: %s", k)
		}
	}
	return opts, nil
}
