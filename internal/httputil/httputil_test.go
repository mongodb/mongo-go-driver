// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package httputil

import (
	"net/http"
	"testing"

	"go.mongodb.org/mongo-driver/v2/internal/assert"
)

type nonDefaultTransport struct{}

func (*nonDefaultTransport) RoundTrip(*http.Request) (*http.Response, error) { return nil, nil }

func TestDefaultHTTPClientTransport(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		client := NewHTTPClient()

		val := assert.ObjectsAreEqual(http.DefaultClient, client)

		assert.True(t, val)
		assert.Equal(t, DefaultHTTPClient, client)
	})

	t.Run("non-default global transport", func(t *testing.T) {
		http.DefaultTransport = &nonDefaultTransport{}

		client := NewHTTPClient()

		val := assert.ObjectsAreEqual(&nonDefaultTransport{}, client.Transport)

		assert.True(t, val)
		assert.Equal(t, DefaultHTTPClient, client)
		assert.NotEqual(t, http.DefaultClient, client) // Sanity Check
	})
}
