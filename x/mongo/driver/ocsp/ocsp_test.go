// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// +build go1.13

package ocsp

import (
	"context"
	"crypto/x509"
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/internal/testutil/assert"
)

func TestOCSP(t *testing.T) {
	t.Run("contactResponders", func(t *testing.T) {
		t.Run("cancelled context is propagated", func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			serverCert := &x509.Certificate{
				OCSPServer: []string{"https://localhost:5000"},
			}
			cfg := config{
				serverCert: serverCert,
				issuer:     &x509.Certificate{},
				cache:      NewCache(),
			}

			_, err := contactResponders(ctx, cfg)
			assert.True(t, errors.Is(err, context.Canceled), "expected error %v, got %v", context.Canceled, err)
		})
	})
}
