// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package conn

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
)

// TLSConfig contains options for configuring an SSL connection to the server.
type TLSConfig struct{ tls.Config }

// NewTLSConfig creates a new TLSConfig.
func NewTLSConfig() *TLSConfig {
	cfg := &TLSConfig{}

	return cfg
}

// SetInsecure sets whether the client should verify the server's certificate chain and hostnames.
func (c *TLSConfig) SetInsecure(allow bool) {
	c.InsecureSkipVerify = allow
}

// AddCaCertFromFile adds a root CA certificate to the configuration given a path to the containing file.
func (c *TLSConfig) AddCaCertFromFile(caFile string) error {
	certBytes, err := loadCert(caFile)
	if err != nil {
		return err
	}

	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return err
	}

	if c.RootCAs == nil {
		c.RootCAs = x509.NewCertPool()
	}

	c.RootCAs.AddCert(cert)

	return nil
}

func loadCert(pemFile string) ([]byte, error) {
	data, err := ioutil.ReadFile(pemFile)
	if err != nil {
		return nil, err
	}

	var certBlock *pem.Block

	for certBlock == nil {
		if data == nil || len(data) == 0 {
			return nil, fmt.Errorf("%s must have both a CERTIFICATE and an RSA PRIVATE KEY section", pemFile)
		}

		block, rest := pem.Decode(data)
		if block == nil {
			return nil, fmt.Errorf("invalid PEM file: %s", pemFile)
		}

		switch block.Type {
		case "CERTIFICATE":
			if certBlock != nil {
				return nil, fmt.Errorf("multiple CERTIFICATE sections in %s", pemFile)
			}

			certBlock = block
		}

		data = rest
	}

	return certBlock.Bytes, nil
}
