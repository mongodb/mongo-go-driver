// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package ocsp

import (
	"bytes"
	"crypto/x509"
	"errors"
)

func getIssuer(peerCert *x509.Certificate, chain []*x509.Certificate) *x509.Certificate {
	for _, cert := range chain {
		// Use RawSubject and RawIssuer rather than cert.Subject.String() and peerCert.Issuer.String because the
		// pkix.Name.String method is not available in Go 1.9.
		if bytes.Equal(cert.RawSubject, peerCert.RawIssuer) {
			return cert
		}
	}
	return nil
}

type config struct {
	serverCert, issuer *x509.Certificate
}

func newConfig(certChain []*x509.Certificate) (config, error) {
	var cfg config

	if len(certChain) == 0 {
		return cfg, errors.New("verified certificate chain contained no certificates")
	}

	cfg.serverCert = certChain[0]
	if len(certChain) == 1 {
		// In the case where the leaf certificate and CA are the same, the chain may only contain one certificate.
		cfg.issuer = certChain[0]
		return cfg, nil
	}

	cfg.issuer = getIssuer(cfg.serverCert, certChain[1:])
	if cfg.issuer == nil {
		return cfg, errors.New("no issuer found for the leaf certificate")
	}
	return cfg, nil
}
