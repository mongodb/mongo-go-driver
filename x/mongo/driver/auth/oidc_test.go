// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package auth

import (
	"regexp"
	"testing"

	"go.mongodb.org/mongo-driver/internal/assert"
)

func TestCreatePatternsForGlobs(t *testing.T) {
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

		check, err := createPatternsForGlobs(hosts)
		assert.NoError(t, err)
		assert.Equal(t,
			[]*regexp.Regexp{
				regexp.MustCompile(`^.*[.]mongodb[.]net(:\d+)?$`),
				regexp.MustCompile(`^.*[.]mongodb-qa[.]net(:\d+)?$`),
				regexp.MustCompile(`^.*[.]mongodb-dev[.]net(:\d+)?$`),
				regexp.MustCompile(`^.*[.]mongodbgov[.]net(:\d+)?$`),
				regexp.MustCompile(`^localhost(:\d+)?$`),
				regexp.MustCompile(`^127[.]0[.]0[.]1(:\d+)?$`),
				regexp.MustCompile(`^::1(:\d+)?$`),
			},
			check,
		)
	})
}
