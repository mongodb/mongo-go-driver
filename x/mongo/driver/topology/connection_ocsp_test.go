// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package topology

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"testing"

	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/mongo/address"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/ocsp"
)

// stubTLSConn is a TLS connection whose handshake always succeeds and which reports no verified certificate chains.
// ocsp.Verify rejects that state, so a connection using this stub succeeds only when OCSP verification is skipped.
type stubTLSConn struct {
	net.Conn
}

func (*stubTLSConn) HandshakeContext(context.Context) error { return nil }

func (*stubTLSConn) ConnectionState() tls.ConnectionState { return tls.ConnectionState{} }

// TestConfigureTLSOCSPVerification asserts when configureTLS performs OCSP verification. Verification is skipped both
// when certificate verification is disabled entirely and when certificate revocation checking is disabled by
// tlsDisableCertificateRevocationCheck.
func TestConfigureTLSOCSPVerification(t *testing.T) {
	var connSource tlsConnectionSourceFn = func(net.Conn, *tls.Config) tlsConn {
		return &stubTLSConn{}
	}

	tests := []struct {
		name                              string
		insecureSkipVerify                bool
		disableCertificateRevocationCheck bool
		wantOCSPErr                       bool
	}{
		{
			name:        "verification runs by default",
			wantOCSPErr: true,
		},
		{
			name:                              "skipped when revocation checking is disabled",
			disableCertificateRevocationCheck: true,
		},
		{
			name:               "skipped when certificate verification is disabled",
			insecureSkipVerify: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ocspOpts := &ocsp.VerifyOptions{Cache: ocsp.NewCache()}

			tlsConfig := &tls.Config{
				InsecureSkipVerify: test.insecureSkipVerify,
				MinVersion:         tls.VersionTLS12,
			}

			conn, err := configureTLS(context.Background(), connSource, nil, address.Address("localhost:27017"),
				tlsConfig, ocspOpts, test.disableCertificateRevocationCheck)

			if test.wantOCSPErr {
				var ocspErr *ocsp.Error
				require.True(t, errors.As(err, &ocspErr), "expected an OCSP verification error, got: %v", err)
				return
			}

			require.NoError(t, err, "expected configureTLS to succeed, got: %v", err)
			require.NotNil(t, conn, "expected a non-nil connection")
		})
	}
}
