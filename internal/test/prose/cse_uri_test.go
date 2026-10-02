// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package prose

import "testing"

func TestCSEContainerURI(t *testing.T) {
	tests := []struct {
		uri  string
		want string
	}{
		{"mongodb://localhost:27017", "mongodb://host.docker.internal:27017/?directConnection=true"},
		{"mongodb://127.0.0.1:27017/?replicaSet=rs0", "mongodb://host.docker.internal:27017/?directConnection=true&replicaSet=rs0"},
		{"mongodb://[::1]:27017", "mongodb://host.docker.internal:27017/?directConnection=true"},
		{"mongodb://localhost", "mongodb://host.docker.internal/?directConnection=true"},
		{"mongodb://[::1]", "mongodb://host.docker.internal/?directConnection=true"},
		{"mongodb://[fd00::1]", "mongodb://[fd00::1]"},
		{"mongodb://localhost:27017/?directConnection=false", "mongodb://host.docker.internal:27017/?directConnection=false"},
		{"mongodb://user:pass@localhost:27017/db", "mongodb://user:pass@host.docker.internal:27017/db?directConnection=true"},
		{"mongodb://db.example.com:27017", "mongodb://db.example.com:27017"},
	}

	for _, tt := range tests {
		t.Run(tt.uri, func(t *testing.T) {
			got, err := cseContainerURI(tt.uri)
			if err != nil {
				t.Fatalf("cseContainerURI(%q) error: %v", tt.uri, err)
			}
			if got != tt.want {
				t.Errorf("cseContainerURI(%q) = %q, want %q", tt.uri, got, tt.want)
			}
		})
	}
}

func TestCSEContainerURIMultipleLoopbackHosts(t *testing.T) {
	if _, err := cseContainerURI("mongodb://localhost:27017,localhost:27018"); err == nil {
		t.Error("expected an error for multiple loopback hosts")
	}
}
