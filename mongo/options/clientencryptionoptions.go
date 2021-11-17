// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

import (
	"crypto/tls"
	"errors"
	"fmt"
)

// ClientEncryptionOptions represents all possible options used to configure a ClientEncryption instance.
type ClientEncryptionOptions struct {
	KeyVaultNamespace string
	KmsProviders      map[string]map[string]interface{}
	TLSConfig         map[string]tls.Config
}

// ClientEncryption creates a new ClientEncryptionOptions instance.
func ClientEncryption() *ClientEncryptionOptions {
	return &ClientEncryptionOptions{}
}

// SetKeyVaultNamespace specifies the namespace of the key vault collection. This is required.
func (c *ClientEncryptionOptions) SetKeyVaultNamespace(ns string) *ClientEncryptionOptions {
	c.KeyVaultNamespace = ns
	return c
}

// SetKmsProviders specifies options for KMS providers. This is required.
func (c *ClientEncryptionOptions) SetKmsProviders(providers map[string]map[string]interface{}) *ClientEncryptionOptions {
	c.KmsProviders = providers
	return c
}

// SetTLSConfig applies custom TLS options to configure connections with each KMS provider.
func (c *ClientEncryptionOptions) SetTLSConfig(tlsOptsMap map[string]map[string]interface{}) (*ClientEncryptionOptions, error) {
	c.TLSConfig = make(map[string]tls.Config)

	for provider, tlsOpts := range tlsOptsMap {
		var cfg tls.Config
		
		for cert := range tlsOpts {
			var err error
			switch cert {
			case "tlsCertificateKeyFile":
				clientCertPath, _ := tlsOpts[cert].(string)
				// apply custom key file password if found, otherwise use empty string
				if keyPwd, found := tlsOpts["tlsCertificateKeyFilePassword"].(string); found {
					_, err = addClientCertFromConcatenatedFile(&cfg, clientCertPath, keyPwd)
				} else {
					_, err = addClientCertFromConcatenatedFile(&cfg, clientCertPath, "")
				}
			case "tlsCertificateKeyFilePassword":
				continue
			case "tlsCAFile":
				CApath, _ := tlsOpts[cert].(string)
				err = addCACertFromFile(&cfg, CApath)
			default:
				return c, errors.New(fmt.Sprintf("Error setting TLS option %v for %v.", cert, provider))
			}
			
			if err != nil {
				return c, err
			}
		}

		c.TLSConfig[provider] = cfg
	}
	return c, nil
}

// MergeClientEncryptionOptions combines the argued ClientEncryptionOptions in a last-one wins fashion.
func MergeClientEncryptionOptions(opts ...*ClientEncryptionOptions) *ClientEncryptionOptions {
	ceo := ClientEncryption()
	for _, opt := range opts {
		if opt == nil {
			continue
		}

		if opt.KeyVaultNamespace != "" {
			ceo.KeyVaultNamespace = opt.KeyVaultNamespace
		}
		if opt.KmsProviders != nil {
			ceo.KmsProviders = opt.KmsProviders
		}
		if opt.TLSConfig != nil {
			ceo.TLSConfig = opt.TLSConfig
		}
	}

	return ceo
}
