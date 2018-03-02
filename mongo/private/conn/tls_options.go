// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package conn

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/asn1"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io/ioutil"
)

// TLSConfig contains options for configuring an SSL connection to the server.
type TLSConfig struct {
	caCert        *x509.Certificate
	clientCert    tls.Certificate
	clientCertSet bool
	insecure      bool
}

// NewTLSConfig creates a new TLSConfig.
func NewTLSConfig() *TLSConfig {
	cfg := &TLSConfig{}

	return cfg
}

// SetInsecure sets whether the client should verify the server's certificate chain and hostnames.
func (c *TLSConfig) SetInsecure(allow bool) {
	c.insecure = allow
}

// AddClientCertFromFile adds a client certificate to the configuration given a path to the
// containing file and returns the certificate's subject name.
func (c *TLSConfig) AddClientCertFromFile(clientFile string) (string, error) {
	data, err := ioutil.ReadFile(clientFile)
	if err != nil {
		return "", err
	}

	cert, err := tls.X509KeyPair(data, data)
	if err != nil {
		return "", err
	}

	c.clientCert = cert
	c.clientCertSet = true

	// The documentation for the tls.X509KeyPair indicates that the Leaf certificate is not
	// retained. Because there isn't any way of creating a tls.Certificate from an x509.Certificate
	// short of calling X509KeyPair on the raw bytes, we're forced to parse the certificate over
	// again to get the subject name.
	certBytes, err := loadCert(data)
	if err != nil {
		return "", err
	}

	crt, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return "", err
	}

	return x509CertSubject(crt), nil
}

// AddCaCertFromFile adds a root CA certificate to the configuration given a path to the containing file.
func (c *TLSConfig) AddCaCertFromFile(caFile string) error {
	data, err := ioutil.ReadFile(caFile)
	if err != nil {
		return err
	}

	certBytes, err := loadCert(data)
	if err != nil {
		return err
	}

	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return err
	}

	c.caCert = cert

	return nil
}

// MakeConfig constructs a new tls.Config from the configuration specified.
func (c *TLSConfig) MakeConfig() *tls.Config {
	cfg := &tls.Config{}

	if c.clientCertSet {
		cfg.Certificates = []tls.Certificate{c.clientCert}
	}

	if c.caCert != nil {
		cfg.RootCAs = x509.NewCertPool()
		cfg.RootCAs.AddCert(c.caCert)
	}
	cfg.InsecureSkipVerify = c.insecure

	return cfg
}

func loadCert(data []byte) ([]byte, error) {
	var certBlock *pem.Block

	for certBlock == nil {
		if data == nil || len(data) == 0 {
			return nil, fmt.Errorf(".pem file must have both a CERTIFICATE and an RSA PRIVATE KEY section")
		}

		block, rest := pem.Decode(data)
		if block == nil {
			return nil, fmt.Errorf("invalid .pem file")
		}

		switch block.Type {
		case "CERTIFICATE":
			if certBlock != nil {
				return nil, fmt.Errorf("multiple CERTIFICATE sections in .pem file")
			}

			certBlock = block
		}

		data = rest
	}

	return certBlock.Bytes, nil
}

// Because the functionality to convert a pkix.Name to a string wasn't added until Go 1.10, we
// need to copy the implementation (along with the attributeTypeNames map below).
func x509CertSubject(cert *x509.Certificate) string {
	r := cert.Subject.ToRDNSequence()

	s := ""
	for i := 0; i < len(r); i++ {
		rdn := r[len(r)-1-i]
		if i > 0 {
			s += ","
		}
		for j, tv := range rdn {
			if j > 0 {
				s += "+"
			}

			oidString := tv.Type.String()
			typeName, ok := attributeTypeNames[oidString]
			if !ok {
				derBytes, err := asn1.Marshal(tv.Value)
				if err == nil {
					s += oidString + "=#" + hex.EncodeToString(derBytes)
					continue // No value escaping necessary.
				}

				typeName = oidString
			}

			valueString := fmt.Sprint(tv.Value)
			escaped := make([]rune, 0, len(valueString))

			for k, c := range valueString {
				escape := false

				switch c {
				case ',', '+', '"', '\\', '<', '>', ';':
					escape = true

				case ' ':
					escape = k == 0 || k == len(valueString)-1

				case '#':
					escape = k == 0
				}

				if escape {
					escaped = append(escaped, '\\', c)
				} else {
					escaped = append(escaped, c)
				}
			}

			s += typeName + "=" + string(escaped)
		}
	}

	return s
}

var attributeTypeNames = map[string]string{
	"2.5.4.6":  "C",
	"2.5.4.10": "O",
	"2.5.4.11": "OU",
	"2.5.4.3":  "CN",
	"2.5.4.5":  "SERIALNUMBER",
	"2.5.4.7":  "L",
	"2.5.4.8":  "ST",
	"2.5.4.9":  "STREET",
	"2.5.4.17": "POSTALCODE",
}
