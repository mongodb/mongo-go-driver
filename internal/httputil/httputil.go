// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package httputil

import (
	"net/http"
)

var DefaultHTTPClient = &http.Client{}

// NewHTTPClient will return the globally-defined DefaultHTTPClient, updating
// the transport if it differs from the http package DefaultTransport.
func NewHTTPClient() *http.Client {
	client := DefaultHTTPClient
	if _, ok := http.DefaultTransport.(*http.Transport); !ok {
		client.Transport = http.DefaultTransport
	}

	return client
}

// CloseIdleHTTPConnections closes any connections which were previously
// connected from previous requests but are now sitting idle in a "keep-alive"
// state. It does not interrupt any connections currently in use.
//
// Borrowed from the Go standard library.
func CloseIdleHTTPConnections(client *http.Client) {
	type closeIdler interface {
		CloseIdleConnections()
	}
	if tr, ok := client.Transport.(closeIdler); ok {
		tr.CloseIdleConnections()
	}
}
