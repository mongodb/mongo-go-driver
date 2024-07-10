// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package auth

import (
	"testing"

	"go.mongodb.org/mongo-driver/internal/assert"
)

func TestGetAllowedHosts(t *testing.T) {
	t.Run("transform allowedHosts patterns", func(t *testing.T) {

		hosts := []string{
			"*.mongodb.net",
			"*.mongodb-qa.net",
			"*.mongodb-dev.net",
			"*.mongodbgov.net",
			"localhost",
			"127.0.0.1",
			"::1",
		}

		assert.Equal(t,
			[]string{
				"^.*[.]mongodb[.]net(:\\d+)?$",
				"^.*[.]mongodb-qa[.]net(:\\d+)?$",
				"^.*[.]mongodb-dev[.]net(:\\d+)?$",
				"^.*[.]mongodbgov[.]net(:\\d+)?$",
				"^localhost(:\\d+)?$",
				"^127[.]0[.]0[.]1(:\\d+)?$",
				"^::1(:\\d+)?$",
			},
			createAllowedHostsPatterns(hosts),
		)
	})
}
