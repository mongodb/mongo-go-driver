// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

import "crypto/tls"

// ClientEncryptionOptions represents all possible options used to configure a ClientEncryption instance.
type ClientEncryptionOptions struct {
	KeyVaultNamespace string
	KmsProviders      map[string]map[string]interface{}
	// TODO: Map string to *tls.Config
	TLSConfig         *tls.Config
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

func (c *ClientEncryptionOptions) SetTLSConfig(tlsOpts map[string]interface{}) (*ClientEncryptionOptions, error) {
	var cfg tls.Config

	if clientCertPath, found := tlsOpts["tlsCertificateKeyFile"].(string); found {
		if keyPwd, found := tlsOpts["keyPwd"].(string); found {
			_, err := addClientCertFromConcatenatedFile(&cfg, clientCertPath, keyPwd)
			if err != nil {
				return c, err
			}
		} else {
			_, err := addClientCertFromConcatenatedFile(&cfg, clientCertPath, "")
			if err != nil {
				return c, err
			}
		}	
	}

	if CApath, found := tlsOpts["tlsCAfile"].(string); found {
		err := addCACertFromFile(&cfg, CApath)
		if err != nil {
			return c , err
		}
	}

	c.TLSConfig = &cfg
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
