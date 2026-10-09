// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

//go:build go1.13

package ocsp

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/httputil"
	"golang.org/x/crypto/ocsp"
)

func TestContactResponders(t *testing.T) {
	t.Run("context cancellation is honored", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		serverCert := &x509.Certificate{
			OCSPServer: []string{"https://localhost:5000"},
		}
		cfg := config{
			serverCert: serverCert,
			issuer:     &x509.Certificate{},
			cache:      NewCache(),
			httpClient: httputil.DefaultHTTPClient,
		}

		res := contactResponders(ctx, cfg)
		assert.Nil(t, res, "expected nil response details, but got %v", res)
	})
	t.Run("context timeout is honored", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		// Create a TCP listener on a random port that doesn't accept any connections, causing
		// connection attempts to hang indefinitely from the client's perspective.
		l, err := net.Listen("tcp", "localhost:0")
		assert.Nil(t, err, "tls.Listen() error: %v", err)
		defer l.Close()

		serverCert := &x509.Certificate{
			OCSPServer: []string{"https://" + l.Addr().String()},
		}
		cfg := config{
			serverCert: serverCert,
			issuer:     &x509.Certificate{},
			cache:      NewCache(),
			httpClient: httputil.DefaultHTTPClient,
		}

		// Expect that contactResponders() returns a nil response but does not cause any errors when
		// the passed-in context times out.
		start := time.Now()
		res := contactResponders(ctx, cfg)
		duration := time.Since(start)
		assert.Nil(t, res, "expected nil response, but got: %v", res)
		assert.True(t, duration <= 5*time.Second, "expected duration to be <= 5s, but was %v", duration)
	})
}

// testOCSPCertificate creates a certificate signed by parent, or self-signed
// when parent is nil.
func testOCSPCertificate(
	t *testing.T,
	serial int64,
	commonName string,
	extKeyUsages []x509.ExtKeyUsage,
	parent *x509.Certificate,
	parentKey *ecdsa.PrivateKey,
) (*x509.Certificate, *ecdsa.PrivateKey) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	assert.Nil(t, err, "GenerateKey error: %v", err)

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(serial),
		Subject:               pkix.Name{CommonName: commonName},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(1 * time.Hour),
		BasicConstraintsValid: true,
		ExtKeyUsage:           extKeyUsages,
	}
	if parent == nil {
		template.IsCA = true
		template.KeyUsage = x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature
		parent = template
		parentKey = key
	}

	der, err := x509.CreateCertificate(rand.Reader, template, parent, &key.PublicKey, parentKey)
	assert.Nil(t, err, "CreateCertificate error: %v", err)
	cert, err := x509.ParseCertificate(der)
	assert.Nil(t, err, "ParseCertificate error: %v", err)
	return cert, key
}

func TestVerifyResponse(t *testing.T) {
	caCert, caKey := testOCSPCertificate(t, 1, "ocsp-test-ca", nil, nil, nil)
	serverCert, serverKey := testOCSPCertificate(t, 2, "ocsp-test-server", nil, caCert, caKey)
	delegateCert, delegateKey := testOCSPCertificate(
		t, 3, "ocsp-test-responder", []x509.ExtKeyUsage{x509.ExtKeyUsageOCSPSigning}, caCert, caKey)

	cfg, err := newConfig([]*x509.Certificate{serverCert, caCert}, &VerifyOptions{Cache: NewCache()})
	assert.Nil(t, err, "newConfig error: %v", err)

	// buildResponse creates a Good response for serverCert with the responder
	// ID taken from responderCert, signed by signerKey, optionally embedding
	// signerCert.
	buildResponse := func(t *testing.T, responderCert, signerCert *x509.Certificate, signerKey *ecdsa.PrivateKey) *ocsp.Response {
		t.Helper()

		template := ocsp.Response{
			Status:       ocsp.Good,
			SerialNumber: serverCert.SerialNumber,
			ThisUpdate:   time.Now().UTC().Add(-1 * time.Minute),
			NextUpdate:   time.Now().UTC().Add(1 * time.Hour),
			Certificate:  signerCert,
		}
		der, err := ocsp.CreateResponse(caCert, responderCert, template, signerKey)
		assert.Nil(t, err, "CreateResponse error: %v", err)
		parsed, err := ocsp.ParseResponseForCert(der, serverCert, caCert)
		assert.Nil(t, err, "ParseResponseForCert error: %v", err)
		return parsed
	}

	t.Run("response signed by issuer accepted", func(t *testing.T) {
		res := buildResponse(t, caCert, nil, caKey)
		err := verifyResponse(cfg, res)
		assert.Nil(t, err, "verifyResponse error: %v", err)
	})
	t.Run("response with embedded issuer certificate accepted", func(t *testing.T) {
		res := buildResponse(t, caCert, caCert, caKey)
		err := verifyResponse(cfg, res)
		assert.Nil(t, err, "verifyResponse error: %v", err)
	})
	t.Run("delegate with OCSP signing EKU accepted", func(t *testing.T) {
		res := buildResponse(t, delegateCert, delegateCert, delegateKey)
		err := verifyResponse(cfg, res)
		assert.Nil(t, err, "verifyResponse error: %v", err)
	})
	t.Run("delegate without EKU rejected", func(t *testing.T) {
		res := buildResponse(t, serverCert, serverCert, serverKey)
		err := verifyResponse(cfg, res)
		assert.NotNil(t, err, "expected verifyResponse error, got nil")
	})
	t.Run("delegate without EKU claiming issuer name rejected", func(t *testing.T) {
		// The responder ID is part of the signed response data, so the signer
		// can claim the issuer's name even though the response was signed with
		// a different certificate.
		res := buildResponse(t, caCert, serverCert, serverKey)
		err := verifyResponse(cfg, res)
		assert.NotNil(t, err, "expected verifyResponse error, got nil")
	})
}
